package golly

import "net/http"

func Boot(f func(Application) error) error {
	a := NewApplication()

	db, err := NewDBConnection(a.Config, a.Name)
	if err != nil {
		return err
	}

	a.DB = db

	return f(a)
}

func Run(mode string) error {
	return Boot(func(a Application) error {
		a.Logger.Infof("Starting App %s (%s)", a.Name, a.Version)

		switch mode {
		case "workers":
		case "web":
			runWeb(a)
		default:
			runWeb(a)
		}
		return nil
	})
}

func runWeb(a Application) {
	http.ListenAndServe(a.Config.GetString("bind"), a)
}
