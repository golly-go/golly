package golly

import (
	"fmt"
	"net/http"
)

func Boot(f func(Application) error) error {
	a := NewApplication()

	db, err := NewDBConnection(a.Config, a.Name)
	if err != nil {
		return err
	}

	a.DB = db

	if err := f(a); err != nil {
		panic(err)
	}

	return nil
}

func (a Application) Run(mode string) error {
	a.Logger.Infof("Starting App %s (%s)", a.Name, a.Version)

	switch mode {
	case "workers":
	case "web":
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
	return http.ListenAndServe(bind, a)
}
