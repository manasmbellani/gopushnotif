# gopushnotif
Golang script to send notification to Pushover application (along with URL screenshots where message provided in "[id] <url>" format) or Sumo

Currently, `gowitness` is used to take screenshot and then send them via pushover. Resolution is kept small, by default, to not exceed Pushover's max attachment limit. 

The input messages are processed by a small number of threads (3, by default) in an attempt to stay under the API limit.

## Examples

Assuming that we have stored the Pushover notification flags in env vars `PUSHOVER_APP_TOKEN` and `PUSHOVER_USER_KEY`, below are some examples of how to use this.

Here is a simple example of sending message `Hello World` via pushover when correct keys are stored in `PUSHOVER_APP_TOKEN` and `PUSHOVER_USER_KEY` env vars:
```
$ echo -e "Hello World." | go run gopushnotif.go
```

If we have the keys stored in CUSTOM env vars such as `CUSTOM_PUSHOVER_APP_TOKEN` and custom `CUSTOM_PUSHOVER_USER_KEY`, then we can use the following command:
```
$ echo -e "Hello World." | go run gopushnotif.go -t $CUSTOM_PUSHOVER_APP_TOKEN -u $CUSTOM_PUSHOVER_USER_KEY
```

If we have the keys stored in AWS Secrets Manager with user key secret as `PUSHOVER_USER_KEY_SECRET` and pushover app token as `PUSHOVER_APP_TOKEN_SECRET`, and we have credentials with appropriate access configured in `~/.aws/credentials` with `aws_profile` profile and region `us-west-2`:
```
$ echo -e "Hello World." | go run gopushnotif.go -puka PUSHOVER_USER_KEY_SECRET -pata PUSHOVER_APP_TOKEN_SECRET -pa -ap aws_profile -ar us-west-2
```

To send message with `[id] <url>` with a screenshot of URL, add a `-p` flag
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p
```

To only perform *dry-run* to test what the script does and not take screenshots and not send notifications, use `-d -v` flags. 
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p -d -v
[test] https://www.google.com
[test2] https://www.msn.com
```

To only send unique messages which have not been sent before, run the command (assuming keys are stored in `PUSHOVER_USER_KEY` and `PUSHOVER_APP_TOKEN`): 
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run gopushnotif.go -su
```

To modify resolution sent to pushover, use `-R` flag. By default, set to `640x480`
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p -d -v
[test] https://www.google.com
[test2] https://www.msn.com
```

## Misc
* TODO: Add support for sending notification by slack
