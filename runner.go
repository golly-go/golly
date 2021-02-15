package golly

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

type RunMode string

var (
	RunModeDefault RunMode = "default"

	RunModeWeb RunMode = "web"

	RunModeWorkers RunMode = "workers"

	AppCommands = []*cobra.Command{
		{
			Use:   "start",
			Short: "Start the web and workers server",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeDefault) },
		},

		{
			Use:   "web",
			Short: "Start the web server",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeWeb) },
		},

		{
			Use:   "workers",
			Short: "Start the workers server",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeWorkers) },
		},

		{
			Use:   "routes",
			Short: "Display the currently defined routes",
			Run: func(cmd *cobra.Command, args []string) {
				Boot(func(a Application) error {
					printRoutes(a.Routes())
					return nil
				})
			},
		},
	}
)

func Run(mode RunMode) {
	if err := Boot(func(a Application) error { a.Run(mode); return nil }); err != nil {
		panic(err)
	}
}

// Seed calls seed for on a function TODO: make this based more on cobra
func Seed(a Application, name string, fn func(Context) error) {
	ctx := context.TODO()

	running := "all"
	if len(os.Args) > 1 {
		running = os.Args[1]
	}

	if running == "list" {
		fmt.Println("\t-\t", name)
	}

	if running == "all" || running == name {

		aCtx := NewContext(ctx)
		aCtx.SetDB(a.DB.Session(&gorm.Session{NewDB: true}))

		if err := fn(aCtx); err != nil {
			a.Logger.Error(err.Error())
			panic(err)
		}
	}
}

func Boot(f func(Application) error) error {
	for _, preboot := range preboots {
		if err := preboot(); err != nil {
			panic(err)
		}
	}

	a := NewApplication()

	db, err := NewDBConnection(a.Config, a.Name)
	if err != nil {
		panic(err)
	}

	a.DB = db

	for _, initializer := range initializers {
		if err := initializer(a); err != nil {
			panic(err)
		}
	}

	if err := f(a); err != nil {
		panic(err)
	}

	return nil
}

func (a Application) Run(mode RunMode) error {
	a.Logger.Infof("Starting App %s (%s)", a.Name, a.Version)

	switch mode {
	case RunModeWorkers:
	case RunModeWeb:
		return runWeb(a)
	default:

		if err := runWeb(a); err != nil {
			return err
		}
	}
	return nil
}

func runWeb(a Application) error {
	var bind string

	if port := a.Config.GetString("port"); port != "" {
		bind = fmt.Sprintf(":%s", port)
	} else {
		bind = a.Config.GetString("bind")
	}

	a.Logger.Infof("Webserver running on %s", bind)

	return http.ListenAndServe(bind, a)
}
