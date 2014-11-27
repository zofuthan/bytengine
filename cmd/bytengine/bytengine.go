package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bitly/go-simplejson"
	"github.com/codegangsta/cli"
	"github.com/gin-gonic/gin"
	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

var Configuration *simplejson.Json
var engine *bytengine.Engine

const (
	VERSION = "0.2.0"
)

func welcomeHandler(ctx *gin.Context) {
	msg := simplejson.New()
	msg.Set("bytengine", "Welcome")
	msg.Set("version", VERSION)
	b, err := msg.MarshalJSON()
	if err != nil {
		fmt.Println(err)
		data := bytengine.ErrorResponse(fmt.Errorf("Error creating welcome message")).JSON()
		ctx.Data(500, "application/json", data)
		return
	}
	ctx.Data(200, "application/json", b)
}

func runScriptHandler(ctx *gin.Context) {
	var form struct {
		Token string `form:"token" binding:"required"`
		Query string `form:"query" binding:"required"`
	}

	ok := ctx.Bind(&form)
	if !ok {
		data := bytengine.ErrorResponse(fmt.Errorf("Missing parameters")).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	r, err := engine.ExecuteScript(form.Token, form.Query)
	if err != nil {
		data := bytengine.ErrorResponse(err).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", r.JSON())
}

func getTokenHandler(ctx *gin.Context) {
	var form struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}
	ok := ctx.Bind(&form)
	if !ok {
		data := bytengine.ErrorResponse(fmt.Errorf("Missing parameters")).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := dsl.NewCommand("login", false)
	cmd.Args["username"] = form.Username
	cmd.Args["password"] = form.Password

	duration, err := Configuration.Get("timeout").Get("authtoken").Int64() // in minutes
	if err != nil {
		data := bytengine.ErrorResponse(fmt.Errorf("Token creation error.")).JSON()
		ctx.Data(400, "application/json", data)
		return
	}
	cmd.Args["duration"] = duration

	r, err := engine.ExecuteCommand("", cmd)
	if err != nil {
		data := bytengine.ErrorResponse(err).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", r.JSON())
}

func getUploadTicketHandler(ctx *gin.Context) {
	var form struct {
		Token    string `form:"token" binding:"required"`
		Database string `form:"database" binding:"required"`
		Path     string `form:"path" binding:"required"`
	}
	ok := ctx.Bind(&form)
	if !ok {
		data := bytengine.ErrorResponse(fmt.Errorf("Missing parameters")).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := dsl.NewCommand("uploadticket", false)
	cmd.Database = form.Database
	cmd.Args["path"] = form.Path

	duration, err := Configuration.Get("timeout").Get("uploadticket").Int64() // in minutes
	if err != nil {
		data := bytengine.ErrorResponse(fmt.Errorf("Token creation error.")).JSON()
		ctx.Data(400, "application/json", data)
		return
	}
	cmd.Args["duration"] = duration

	r, err := engine.ExecuteCommand(form.Token, cmd)
	if err != nil {
		data := bytengine.ErrorResponse(err).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", r.JSON())
}

func uploadFileHelper(max int, ctx *gin.Context) (string, int, error) {
	total := 0             // total bytes read/written
	maxbytes := 1024 * max // maximum upload size in bytes

	tmpfile, err := ioutil.TempFile("", "bytengine_upload_") // upload file name from header
	if err != nil {
		return "", 0, err
	}
	defer tmpfile.Close()

	// create read buffer
	var bsize int64 = 16 * 1024 // 16 kb
	buffer := make([]byte, bsize)

	// get stream
	mr, err := ctx.Request.MultipartReader()
	if err != nil {
		return "", 0, err
	}
	in_f, err := mr.NextPart()
	if err != nil {
		return "", 0, err
	}
	defer in_f.Close()

	// start reading/writing

	for {
		// read
		n, err := in_f.Read(buffer)
		if n == 0 {
			break
		}
		if err != nil {
			return "", 0, err
		}
		// update total bytes
		total += n
		if total > maxbytes {
			return "", 0, fmt.Errorf("exceeded maximum file size of %d bytes", maxbytes)
		}
		// write
		n, err = tmpfile.Write(buffer[:n])
		if err != nil {
			return "", 0, err
		}
	}

	return tmpfile.Name(), total, nil
}

func uploadFileHandler(ctx *gin.Context) {
	ticket := ctx.Params.ByName("ticket")
	filename, _, err := uploadFileHelper(300, ctx)
	if err != nil {
		data := bytengine.ErrorResponse(fmt.Errorf("upload failed: %s", err.Error())).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := dsl.NewCommand("writebytes", false)
	cmd.Args["ticket"] = ticket
	cmd.Args["tmpfile"] = filename

	r, err := engine.ExecuteCommand("", cmd)
	if err != nil {
		data := bytengine.ErrorResponse(err).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", r.JSON())
}

func downloadFileHandler(ctx *gin.Context) {
	var form struct {
		Token    string `form:"token" binding:"required"`
		Database string `form:"database" binding:"required"`
		Path     string `form:"path" binding:"required"`
	}
	ok := ctx.Bind(&form)
	if !ok {
		data := bytengine.ErrorResponse(fmt.Errorf("Missing parameters")).JSON()
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := dsl.NewCommand("readbytes", false)
	cmd.Database = form.Database
	cmd.Args["path"] = form.Path
	cmd.Args["writer"] = ctx.Writer

	ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
	_, err := engine.ExecuteCommand(form.Token, cmd)
	if err != nil {
		data := bytengine.ErrorResponse(err).String()
		ctx.String(400, data)
		return
	}
}

func directaccessHandler(ctx *gin.Context) {
	db := ctx.Params.ByName("database")
	path := ctx.Params.ByName("path")
	layer := ctx.Params.ByName("layer")

	cmd := dsl.NewCommand("directaccess", false)
	cmd.Args["database"] = db
	cmd.Args["path"] = path
	cmd.Args["layer"] = layer
	cmd.Args["writer"] = ctx.Writer

	if layer == "json" {
		ctx.Writer.Header().Set("Content-Type", "application/json")
	} else {
		ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
	}

	_, err := engine.ExecuteCommand("", cmd)
	if err != nil {
		data := bytengine.ErrorResponse(err).String()
		ctx.String(404, data)
		return
	}
}

func main() {
	app := cli.NewApp()
	engine = bytengine.NewEngine()

	createadminCmd := cli.Command{
		Name: "createadmin",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "u", Value: "", Usage: "username"},
			cli.StringFlag{Name: "p", Value: "", Usage: "password"},
			cli.StringFlag{Name: "c", Value: "config.json"},
		},
		Action: func(c *cli.Context) {
			usr := c.String("u")
			pw := c.String("p")
			pth := c.String("c")

			rdr, err := os.Open(pth)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}
			Configuration, err = simplejson.NewFromReader(rdr)

			// start bytengine
			engine.Start(Configuration.Get("bytengine"))

			err = engine.CreateAdminUser(usr, pw)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}
			fmt.Println("...done")
		},
	}
	run := cli.Command{
		Name: "run",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "c", Value: "config.json"},
		},
		Action: func(c *cli.Context) {
			// get configuration file/info
			pth := c.String("c")
			rdr, err := os.Open(pth)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}
			Configuration, err = simplejson.NewFromReader(rdr)
			addr := Configuration.Get("address").MustString()
			port := Configuration.Get("port").MustInt()

			// start bytengine
			engine.Start(Configuration.Get("bytengine"))

			// setup routes
			router := gin.Default()
			router.GET("/", welcomeHandler)
			router.POST("/bfs/query", runScriptHandler)
			router.POST("/bfs/token", getTokenHandler)
			router.POST("/bfs/uploadticket", getUploadTicketHandler)
			router.POST("/bfs/writebytes/:ticket", uploadFileHandler)
			router.POST("/bfs/readbytes", downloadFileHandler)
			router.GET("/bfs/direct/:layer/:database/*path", directaccessHandler)

			router.Run(fmt.Sprintf("%s:%d", addr, port))
		},
	}
	app.Commands = []cli.Command{createadminCmd, run}
	app.Run(os.Args)
}
