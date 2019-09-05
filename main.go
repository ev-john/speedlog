package main

import (
	"errors"
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/gorilla/mux"
	"github.com/khyurri/speedlog/engine"
	"github.com/khyurri/speedlog/engine/mongo"
	"github.com/khyurri/speedlog/plugins"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const defaultTimezone = "UTC-0"

type params struct {
	Mode        string `arg:"positional" help:"Available modes: runserver, adduser, addproject. Default: runserver"`
	Mongo       string `arg:"-d" help:"Mode runserver. Mongodb url. Default 127.0.0.1:27017"`
	JWTKey      string `arg:"-j" help:"Mode runserver. JWT secret key."`
	AllowOrigin string `arg:"-o" help:"Mode runserver. Add Access-Control-Allow-Origin header with passed by param value"`
	TZ          string `arg:"-t" help:"Mode runserver. Timezone. Default UTC±00:00."`
	Graphite    string `arg:"-g" help:"Mode runserver. Graphite host:port"`
	Project     string `arg:"-r" help:"Modes runserver, addproject. Project title."`
	Login       string `arg:"-l" help:"Mode adduser. Login for new user"`
	Password    string `arg:"-p" help:"Mode adduser. Password for new user"`
}

func parseTZ(timezone string) (*time.Location, error) {
	if timezone == defaultTimezone {
		return time.FixedZone(timezone, 0), nil
	}
	return time.LoadLocation(timezone)
}

func addProjectMode(cliParams *params, dbEngine mongo.DataStore) (err error) {
	if len(cliParams.Project) > 0 {
		err = dbEngine.AddProject(cliParams.Project)
	} else {
		err = errors.New("--project param not found")
	}
	return
}

func ok(lg *log.Logger, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		lg.Fatalf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
	}
}

func main() {

	cliParams := &params{}

	////////////////////////////////////////
	//
	// DEFAULTS
	cliParams.Mode = "runserver"
	cliParams.Mongo = "127.0.0.1:27017"
	cliParams.TZ = defaultTimezone
	//
	////////////////////////////////////////

	arg.MustParse(cliParams)
	cLogger := log.New(os.Stdout, "speedlog ", log.LstdFlags|log.Lshortfile)

	dbEngine, err := mongo.New("speedlog", cliParams.Mongo)
	ok(cLogger, err)
	defer dbEngine.Session.Close()

	location, err := parseTZ(cliParams.TZ)
	ok(cLogger, err)

	env := engine.NewEnv(dbEngine, cliParams.JWTKey, location)
	if len(cliParams.AllowOrigin) > 0 {
		env.AllowOrigin = cliParams.AllowOrigin
	}
	switch cliParams.Mode {
	case "runserver":

		if len(cliParams.JWTKey) == 0 {
			cLogger.Printf("[error] cannot start server. Required jwtkey")
			return
		}

		if len(cliParams.Graphite) > 0 {
			graphite := plugins.NewGraphite(cliParams.Graphite, location)
			graphite.Load(dbEngine)
		}

		if len(cliParams.Project) > 0 {
			err = dbEngine.AddProject(cliParams.Project)
			if err != nil {
				cLogger.Printf("[info] project %s exists. skipping", cliParams.Project)
			}
		}

		r := mux.NewRouter()
		env.ExportEventRoutes(r)
		env.ExportUserRoutes(r)
		env.ExportProjectRoutes(r)

		srv := &http.Server{
			Handler:      r,
			Addr:         ":8012",
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}
		err = srv.ListenAndServe()
		ok(cLogger, err)
	case "adduser":
		err := env.DBEngine.AddUser(cliParams.Login, cliParams.Password)
		ok(cLogger, err)
	case "addproject":
		err = addProjectMode(cliParams, dbEngine)
		ok(cLogger, err)
	default:
		ok(cLogger, errors.New(fmt.Sprintf("unknown mode '%s'", cliParams.Mode)))
	}

}
