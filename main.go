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

var debug bool
var verbose bool
var configDir string
var configFile string
var version string

// cacheDir is separately configurable, as on some systems you want it to write to /tmp so that the keys are purged
// after the system is rebooted.
var cacheDir string

func main() {
	var targetRole string
	var baseProfile string
	var oneTimePasscode string
	var tokenNeedsRefresh bool
	var showVersionInfo bool
	var showRoleList bool

	flag.BoolVar(&debug, "debug", false, "Enables debug logging.")
	flag.BoolVar(&showRoleList, "list", false, "Displays a list of configured roles, then exits.")
	flag.BoolVar(&showVersionInfo, "version", false, "Show version information.")
	flag.BoolVar(&tokenNeedsRefresh, "refresh", false, "Force a refresh of all tokens")
	flag.BoolVar(&verbose, "verbose", false, "Enables verbose logging.")
	flag.StringVar(&baseProfile, "profile", "", "The base AWS config profile to use when creating the session.")
	flag.StringVar(&oneTimePasscode, "code", "", "MFA Token OTP - The 6+ digit code that refreshes every 30 seconds.")
	flag.StringVar(&targetRole, "role", "", "The role name or alias to assume.")

	flag.Parse()

	config.Debug, config.Verbose = debug, verbose

	if showVersionInfo {
		if version == "" {
			fmt.Println("Roo version unknown - 'version' variable not set at build time.")
		} else {
			fmt.Println("Roo version", version)
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
		fmt.Println("Error: Please provide the MFA Token code (OTP) via the '-code' parameter.")
		os.Exit(1)
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

	if debug {
		log.Println("We're going to want to run the following command:", flag.Args())
	}
	cmd := exec.Command("/usr/bin/env", flag.Args()...)

	// Attach STDOUT, STDERR, and STDIN
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

	err = cmd.Run()
	if err != nil {
		log.Println("An error occurred while trying to execute command:", err)
	}
}
