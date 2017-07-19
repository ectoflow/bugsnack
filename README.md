Bugsnack
-----

Bugsnack provides an easy way to track errors by injecting an
`bugsnack.ErrorReporter` throughout your application, then 
providing a concrete `ErrorReporter` implementation in `main.main`.

At fromAtoB, we use `bugsnag` extensively to track unhandled errors 
throughout our services - `bugsnack` is a simple, idiomatic client for 
sending errors from Go to `bugsnag` and similar services, while easily
being swapped out for development/local environments and mocked in 
CI setups.

In the future, it would be straightforward to expand `bugsnack` to work
with other "crash reporting" services, like `sentry`, `airbrake`, or `rollbar`.
Contributions to do this are welcome! (just send a PR)

# Basic Usage

However, try to not pass a `bugsnack.ErrorReporter` any deeper than absolutely
neccessary, as it's far better to simply return errors and pass them 
up the call stack until you *must* deal with it somehow. Good use cases would be
dealing with errors that rise in a long-running goroutine that shouldn't abort, or 
wrapping panics in a `http.Handler`. A bad use would be firing off `Report(ctx, err)` 
every time you check for an error within your program.

Once you've determined where you should be using a `bugsnack.ErrorReporter`, simply
pass it from `main.main` to the places it's needed, and then use it as follows:

```go
func Work(er bugsnack.ErrorReporter) {
    for {
        _, err := DoSomethingThatMightBreak()
        if err != nil {
            er.Report(context.TODO(), err)
            continue
        }
        time.Sleep(time.Second)
    }
}
```

then from your `main.main`:

```go
func main() {
    // send to bugsnag example
    //go Work(&bugsnack.BugsnagReporter{
    //    APIKey: "put your api key here",
    //    ReleaseStage: "production",
    //    Doer: http.DefaultClient,
    //    Backup: &bugsnack.WriterReporter{Writer: os.Stdout},
    //})
    go Work(&bugsnack.WriterReporter{Writer: os.Stdout})
    // do something blocking while the background routine runs
}
```

# Metadata Support

You may provide optional `ErrorClass`, `Context`, `GroupingHash`, `Severity` and arbitrary `EventMetadata`:

```go
import "runtime"

func Work(er bugsnack.ErrorReporter) {
    for {
        _, err := DoSomethingThatMightBreak()
        if err != nil {
            er.Report(context.TODO(), err, &bugsnack.BugsnagMetadata{
                ErrorClass: "network.timeout",
                Context: "fetchWorker",
                GroupingHash: "timeouts", // https://docs.bugsnag.com/product/error-grouping/#custom-grouping-hash
                Severity: "info",
                EventMetadata: &map[string]interface{}{
                    "data": map[string]interface{}{
                        "os": runtime.GOOS,
                    },
                    "key1": "value1",
                    "key2": "value2",
                    "arbitraryData": map[string]interface{}{
                        "goVersion": runtime.Version(),
                        "nested": map[string]interface{}{
                            "nestedKey": "value",
                        },
                    },
                },
            })
            continue
        }
        time.Sleep(time.Second)
    }
}
```

Please follow docs at https://docs.bugsnag.com/api/error-reporting/#json-payload


# Advanced Usage

Bugsnack includes some more exotic reporter-chaining functions, such as 
`bugsnack.MultiReporter`, which can send the same error to multiple 
`bugsnack.ErrorReporter`s at the same time. Chaining `bugsnack.ErrorReporter`s 
is a powerful way to ensure you track all the errors within your application.

# LICENSE

MIT, see LICENSE
