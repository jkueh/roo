# roo

A little utility to help make AWS calls from an authentication account with MFA.

It caches credentials in its 'pouch' (aka a file on disk), reaching out to renew them when needed.

## Use Case

I wrote this because I came across the following pattern:

* Every user logs into an 'authentication account' as an IAM user.  
  Each user can create their own set of access keys, and they _must_ have MFA enabled in order to log in.
* Once they have logged in to the authentication account, users then assume a role in another account to do what they
  need to do (e.g. deleting the production environment).

Quite frankly, I tired of using a convoluted bash script to generate session tokens in order to assume a role, so I
wrote a thing to do it for me.

## Command Precedence

There's a few flags that are used to determine behaviour, and they are given the following precedence:

* `--console` or `--console-url`
* `--write-profile`

If none of the above are specified, it defaults to executing the command provided.

## Configuration

If you run `roo` once without a configuration file, it will generate a dummy one for you (at `${HOME}/.roo/config.yaml`)

Alternatively, you can write your own (See Configuration Reference).

### Configuration Reference

This is an example of the configuration file, commonly found at `${HOME}/.roo/config.yaml`.
Modify these values to your liking / requirements

```yaml
mfa_serial: arn:aws:iam::000000000000:mfa/my_mfa_serial
base_profile: some-base-profile # optional - this is the AWS profile you use to log into the authentication account.
roles:
  - name: something-prod-readonly
    default: yes # Optional, but helpful!
    arn: arn:aws:iam::000000000000:role/ReadOnly
    aliases: # Optional, and also helpful!
      - something-prod
      - prod-readonly
    # target_aws_profile (Optional):
    # This is the name of the profile that 'roo' will write to when '-write-profile' is specified on the command line.
    # If not specified, using -write-profile will require -profile-target.
    target_aws_profile: "roo-default"

  - name: something-prod-deleteonly
    arn: arn:aws:iam::000000000000:role/DeleteOnly
    aliases:
      - deleteprod
    target_aws_profile: "my-other-profile"

  - name: something-test-developer
    arn: arn:aws:iam::111111111111:role/Developer
    aliases:
      - something-dev
      - test-dev
    target_aws_profile: "yet-another-profile"
```
