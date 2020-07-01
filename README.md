# gopushnotif
Golang script to send notification to Pushover application (along with URL screenshots where message provided in "[id] <url>" format)

Currently, `gowitness` is used to take screenshot and then send them via pushover. Resolution is kept small, by default, to not exceed Pushover's max attachment limit. 

The input messages are processed by a small number of threads (3, by default) in an attempt to stay under the API limit.

## Examples

Assuming that we have stored the Pushover notification flags in env vars `PUSHOVER_APP_TOKEN` and `PUSHOVER_USER_KEY`, below are some examples of how to use this.

Here is a simple example of sending message `Hello World` via pushover. 
```
$ echo -e "Hello World." | go run /opt/athena-tools/notify/gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY
```

To send message with `[id] <url>` with a screenshot of URL, add a `-p` flag
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run /opt/athena-tools/notify/gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p
```

To only perform *dry-run* to test what the script does and not take screenshots and not send notifications, use `-d -v` flags. 

```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run /opt/athena-tools/notify/gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p -d -v
[test] https://www.google.com
[test2] https://www.msn.com
```

To modify resolution sent to pushover, use `-R` flag. By default, set to `640x480`
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run /opt/athena-tools/notify/gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p -d -v
[test] https://www.google.com
[test2] https://www.msn.com
```

## Misc
* TODO: Add support for sending notification by slack