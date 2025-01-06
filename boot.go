package golly

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func boot(f func(*Application) error) error {
	a := &Application{}

	// a := NewApp()

	signals(a)

	if err := a.initialize(); err != nil {
		return err
	}

	// defer a.Shutdown(NewContext(a.context))

	// a.Logger.Infof("Good golly were booting %s (%s)", a.Name, a.Version)

	if err := f(a); err != nil {
		return err
	}

	return nil
}

func signals(app *Application) {
	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(c <-chan os.Signal) {
		<-c
		// app.Logger.Infof("issuing shutdown due to signal (%s)", signal.String())
		// app.Shutdown(NewContext(app.context))
	}(sig)
}

// Run application lifecycle
func Run(fn GollyAppFunc) {
	if err := boot(fn); err != nil {
		fmt.Printf("Application Error: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
