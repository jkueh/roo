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

## Configuration

Create a file at `~/.roo/config.yaml`, with the following contents (modify values to your liking / requirements)

```yaml
mfa_serial: arn:aws:iam::000000000000:mfa/my_mfa_serial
roles:
  - name: something-prod-readonly
    arn: arn:aws:iam::000000000000:role/ReadOnly
    aliases:
      - something-prod
      - prod-readonly

  - name: something-prod-deleteonly
    arn: arn:aws:iam::000000000000:role/DeleteOnly
    aliases:
      - deleteprod

  - name: something-test-developer
    arn: arn:aws:iam::111111111111:role/Developer
    aliases:
      - something-dev
      - test-dev
```
