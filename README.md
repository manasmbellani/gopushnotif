# gopushnotif
Golang script to send notification to Pushover application (along with URL screenshots where message provided in "[id] <url>" format)

Currently, `gowitness` is used to take screenshot and all screenshots are stored in folder `out-screenshots`. Plan is to make this run concurrently in near-future.

## Examples

Assuming that we have stored the Pushover notification flags in env vars `PUSHOVER_APP_TOKEN` and `PUSHOVER_USER_KEY`, below are some examples of how to use this.

Here is a simple example of sending message `Hello World` via pushover. 
```
$ echo -e "Hello World." | go run /opt/athena-tools/notify/gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p
```

To send message with `[id] <url>` with a screenshot of URL, add a `-p` flag
```
$ echo -e "[test] https://www.google.com\n[test2] https://www.msn.com" | go run /opt/athena-tools/notify/gopushnotif.go -t $PUSHOVER_APP_TOKEN -u $PUSHOVER_USER_KEY -p
```
