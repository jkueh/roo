package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/jkueh/roo/cachedcredsprovider"

	"github.com/jkueh/roo/config"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

const rooVersion = "0.2.0"

var debug bool
var verbose bool
var configDir string
var configFile string
var version string

// cacheDir is separately configurable, as on some systems you want it to write to /tmp so that the keys are purged
// after the system is rebooted.
var cacheDir string

func main() {
	var baseProfile string
	var oneTimePasscode string
	var showRoleList bool
	var showVersionInfo bool
	var targetRole string
	var tokenNeedsRefresh bool
	var writeToProfile bool
	var targetProfile string

	flag.BoolVar(&debug, "debug", false, "Enables debug logging.")
	flag.BoolVar(&showRoleList, "list", false, "Displays a list of configured roles, then exits.")
	flag.BoolVar(&showVersionInfo, "version", false, "Show version information.")
	flag.BoolVar(&tokenNeedsRefresh, "refresh", false, "Force a refresh of all tokens")
	flag.BoolVar(&verbose, "verbose", false, "Enables verbose logging.")
	flag.StringVar(&baseProfile, "profile", "", "The base AWS config profile to use when creating the session.")
	flag.StringVar(&oneTimePasscode, "code", "", "MFA Token OTP - The 6+ digit code that refreshes every 30 seconds.")
	flag.StringVar(&targetRole, "role", "", "The role name or alias to assume.")
	flag.BoolVar(
		&writeToProfile,
		"write-profile",
		false,
		"If set, roo will write the credentials to an AWS profile using the AWS CLI.",
	)
	flag.StringVar(&targetProfile, "target-profile", "", "The name of the profile to write credentials for.")

	flag.Parse()

	config.Debug, config.Verbose = debug, verbose

	if showVersionInfo {
		if verbose {
			fmt.Println("Roo version", rooVersion)
		} else {
			fmt.Println(rooVersion)
		}
		os.Exit(0)
	}

	// Some flag debugging
	if debug {
		log.Println("Command Line Parameters")
		flag.VisitAll(func(f *flag.Flag) {
			log.Println(f.Name+":", "\t", f.Value)
		})
		log.Println()
	}

	// Ensure we have a role definition for the role
	conf := config.New(configFile)

	if showRoleList {
		conf.ListRoles()
		os.Exit(0)
	}

	var role *config.RoleConfig
	if targetRole == "" {
		// See if we can pull a default
		role = conf.GetDefaultRole()
		if role == nil {
			flag.Usage()
			log.Fatalln("Role not provided (-role)")
		}
	} else {
		role = conf.GetRole(targetRole)
	}

	if debug {
		log.Println("role:", role)
	}

	if role.ARN == "" {
		log.Fatalln("Unable to find role by name or alias:", targetRole)
	}

	// If a base profile wasn't specified on the command line, then try use a default - if configured.
	if baseProfile == "" {
		baseProfile = conf.DefaultProfile
	}

	// The cache file name we use is {{.AccountNumber}}-{{.RoleName}}.json
	accountNumberRE := regexp.MustCompile("arn:aws:iam::([0-9]+)")
	accountNumberMatch := accountNumberRE.FindStringSubmatch(role.ARN)
	if len(accountNumberMatch) == 0 {
		log.Fatalln("Unable to determine account number from ARN.")
	}
	accountNumber := accountNumberMatch[1]

	roleNameRE := regexp.MustCompile(":role/(.*)")
	roleNameMatch := roleNameRE.FindStringSubmatch(role.ARN)
	if len(roleNameMatch) == 0 {
		log.Fatalln("Unable to determine the role name from ARN.")
	}
	roleName := roleNameMatch[1]

	if debug {
		log.Println("Account Number:", accountNumber)
		log.Println("Role Name:     ", roleName)
	}

	cacheFileName := fmt.Sprintf("%s-%s.gob", accountNumber, roleName)
	cacheFilePath := strings.Join([]string{cacheDir, cacheFileName}, string(os.PathSeparator))

	// Define our credential providers
	cachedProvider := cachedcredsprovider.New(cacheFilePath)
	if debug {
		currentCreds, _ := cachedProvider.Retrieve()
		if currentCreds.AccessKeyID != "" {
			log.Println("Current Access Key ID:", currentCreds.AccessKeyID)
		}
	}

	if !tokenNeedsRefresh {
		tokenNeedsRefresh = cachedProvider.IsExpired()
		if debug && tokenNeedsRefresh {
			log.Println("Refresh required - cachedProvider indicated credentials are expired.")
		}
	}

	if tokenNeedsRefresh && oneTimePasscode == "" {
		oneTimePasscodePrompts := 0
		oneTimePasscodeValid := false
		var oneTimePasscodeValidationError error
		for oneTimePasscodePrompts < 3 && oneTimePasscodeValid == false {
			oneTimePasscodeInput := getStringInputFromUser("MFA Code")

			// Ensure that trailing newline characters are removed (e.g. Windows will add \r at the end)
			oneTimePasscode = strings.TrimRight(oneTimePasscodeInput, "\r\n")

			if debug {
				log.Println("MFA Code Provided:", oneTimePasscode)
			}
			oneTimePasscodeValid, oneTimePasscodeValidationError = oneTimePasscodeIsValid(oneTimePasscode)
			if oneTimePasscodeValidationError != nil {
				log.Println("Invalid MFA Code:", oneTimePasscodeValidationError)
			}
			oneTimePasscodePrompts++
		}
		if !oneTimePasscodeValid {
			fmt.Println("Error: Please provide the MFA Token code (OTP) via the '-code' parameter.")
			os.Exit(1)
		}
	}

	// The static credentials we'll use to build the targetRole session

	// At this point - Work out if we need to load the initial credentials for the authentication account, or if we can
	// jump straight to exporting the existing tokens.
	if tokenNeedsRefresh {
		// Time to do the hard work - Get to the point where we can cache credentials to disk.
		authAccountSessionOpts := session.Options{}
		if baseProfile != "" {
			authAccountSessionOpts.Profile = baseProfile
		}
		authAccountSession, err := session.NewSessionWithOptions(authAccountSessionOpts)
		if err != nil {
			log.Fatalln("Unable to create a session for the authentication account:", err)
		}
		stsClient := sts.New(authAccountSession)
		callerIdentityOutput, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			log.Fatalln("An error occurred while trying to get caller identity:", err)
		}
		if verbose {
			log.Println("Hello world, I'm", *callerIdentityOutput.Arn, "- Time to assume another role!")
		}
		timeNow := time.Now()
		timeNowUnixNanoString := strconv.FormatInt(timeNow.UnixNano(), 10)
		assumeRoleInput := &sts.AssumeRoleInput{
			SerialNumber:    aws.String(conf.MFASerial),
			TokenCode:       aws.String(oneTimePasscode),
			RoleArn:         aws.String(role.ARN),
			RoleSessionName: aws.String("roo-" + timeNowUnixNanoString),
		}
		assumeRoleOutput, err := stsClient.AssumeRole(assumeRoleInput)
		if err != nil {
			log.Fatalln("An error occurred while trying to assume the target role:", err)
		}
		if verbose {
			log.Println("We have successfully assumed the role:", *assumeRoleOutput.AssumedRoleUser.Arn)
		}

		// Okay, time to build the cachedCredentials object.
		err = cachedProvider.WriteNewCredentialsFromSTS(assumeRoleOutput.Credentials, cacheFilePath)
		if err != nil {
			log.Println("WARNING: An error occurred while trying to write new credentials to the cache file:", err)
		} else {
			if debug {
				log.Println("New credentials written - New Access Key ID:", *assumeRoleOutput.Credentials.AccessKeyId)
			}
		}
	} else {
		if verbose {
			log.Println("Using cached credentials!")
		}
	}

	// Now we use the new cachedProvider to export our environment variables
	retrievedCreds, err := cachedProvider.Retrieve()
	if err != nil {
		log.Fatalln("Unable to retrieve credentials:", err)
	}
	if debug {
		log.Println("Retrieved credentials with Access Key ID", retrievedCreds.AccessKeyID)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", retrievedCreds.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", retrievedCreds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", retrievedCreds.SessionToken)

	if debug {
		staticCredentials := credentials.NewStaticCredentialsFromCreds(retrievedCreds)
		stsSession := session.Must(session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Credentials: staticCredentials,
			},
		}))

		stsClient := sts.New(stsSession)

		callerIdentityOutput, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			log.Fatalln(
				"An error occurred while trying to get caller identity when working out who we have credentials for:",
				err,
			)
		}
		if verbose {
			log.Println("Hello world, I'm", *callerIdentityOutput.Arn)
		}
	}

	// There is an exception to evaluating commands - And that's if we've been asked to write these credentials to file.
	if writeToProfile {

		var targetProfileName string // We'll need to coalesce through some config , combined with flags.

		if targetProfile == "" && role.TargetAWSProfile == "" {
			log.Println("Please specify a target profile with -target-profile, or by specifying it in the config file.")
			os.Exit(1)
		}

		if targetProfile != "" {
			targetProfileName = targetProfile
		} else {
			targetProfileName = role.TargetAWSProfile
		}

		if verbose {
			log.Println("We're going to write to profile! The profile's named", targetProfileName)
		}
		// Step 0 is to check that we have the AWS executable somewhere in the PATH.
		cliPath, err := exec.LookPath("aws")
		if err != nil {
			log.Fatalln("Unable to find AWS CLI executable in PATH:", err)
		}
		if debug {
			log.Println("Found it!", cliPath)
		}

		// Okay, time to execute the commands we need to execute.
		baseCommand := []string{cliPath, "--profile", targetProfileName, "configure", "set"}

		// Apologies for all the spread operators...

		// Set: Access Key ID
		err = executeCommand(append(baseCommand, []string{"aws_access_key_id", retrievedCreds.AccessKeyID}...)...)
		if err != nil {
			log.Println("An error occurred while trying to write the aws_access_key_id to file:", err)
		}

		// Set: Secret Access Key
		err = executeCommand(append(baseCommand, []string{"aws_secret_access_key", retrievedCreds.SecretAccessKey}...)...)
		if err != nil {
			log.Println("An error occurred while trying to write the aws_secret_access_key to file:", err)
		}

		// Set: Session Token
		err = executeCommand(append(baseCommand, []string{"aws_session_token", retrievedCreds.SessionToken}...)...)
		if err != nil {
			log.Println("An error occurred while trying to write the aws_access_key_id to file:", err)
		}

		// Set: Expiration Timestamp - Not used by the AWS CLI, but will allow the user to check.
		err = executeCommand(append(baseCommand, []string{
			"expiration_time", cachedProvider.GetCredentialExpiryTime().String()}...)...,
		)
		if err != nil {
			log.Println("An error occurred while trying to write the aws_access_key_id to file:", err)
		}

		fmt.Println("Profile written:", targetProfileName)
	} else {
		if debug {
			log.Println("flag.Args() length:", len(flag.Args()))
		}
		if len(flag.Args()) == 0 { // Let's make sure we have something to run here...
			// println() for STDERR output
			println("Please provide a command to execute, e.g.:")
			println("roo -role my_role_name aws sts get-caller-identity")
			os.Exit(100)
		}
		if debug {
			log.Println("We're going to want to run the following command:", flag.Args())
		}
		err := executeCommand(flag.Args()...)
		if err != nil {
			log.Println("An error occurred while trying to execute command:", err)
		}
	}
}
