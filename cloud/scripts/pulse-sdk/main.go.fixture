package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "time"
 "github.com/getsentry/sentry-go"
)
func main() {
 data,err:=os.ReadFile(os.Getenv("PULSE_TEST_HANDOFF"));if err!=nil{panic(err)}
 var cfg struct{DSN string `json:"dsn"`};if err=json.Unmarshal(data,&cfg);err!=nil{panic(err)}
 if err=sentry.Init(sentry.ClientOptions{Dsn:cfg.DSN,EnableTracing:false,SendDefaultPII:false});err!=nil{panic(err)}
 sentry.WithScope(func(s *sentry.Scope){s.SetContext("note",sentry.Context{"text":"PRIVATE_GO_NOTE_SENTINEL"});s.SetUser(sentry.User{Email:"go-sentinel@audit.invalid"});sentry.CaptureException(errors.New("Local Go SDK exception"))})
 sentry.CaptureMessage("Local Go SDK message")
 if !sentry.Flush(10*time.Second){panic("local Go SDK flush failed")}
 fmt.Println("Official Go SDK: flushed exception and message")
}
