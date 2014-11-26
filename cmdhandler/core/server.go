package core

import (
	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: server.listdb
func ServerListDb(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return bytengine.FileSystemPlugin.ListDatabase(filter)
}

// handler for: server.newdb
func ServerNewDb(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	db := cmd.Args["database"].(string)
	return bytengine.FileSystemPlugin.CreateDatabase(db)
}

// handler for: server.init
func ServerInit(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	return bytengine.FileSystemPlugin.ClearAll()
}

// handler for: server.dropdb
func ServerDropDb(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	db := cmd.Args["database"].(string)
	return bytengine.FileSystemPlugin.DropDatabase(db)
}

func init() {
	bytengine.RegisterCommandHandler("server.listdb", ServerListDb)
	bytengine.RegisterCommandHandler("server.newdb", ServerNewDb)
	bytengine.RegisterCommandHandler("server.init", ServerInit)
	bytengine.RegisterCommandHandler("server.dropdb", ServerDropDb)
}