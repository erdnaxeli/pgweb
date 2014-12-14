package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var MIME_TYPES = map[string]string{
	".css":  "text/css",
	".js":   "application/javascript",
	".icon": "image-x-icon",
	".eot":  "application/vnd.ms-fontobject",
	".svg":  "image/svg+xml",
	".ttf":  "application/font-sfnt",
	".woff": "application/font-woff",
}

type Error struct {
	Message string `json:"error"`
}

func NewError(err error) Error {
	return Error{err.Error()}
}

func assetContentType(name string) string {
	mime := MIME_TYPES[filepath.Ext(name)]

	if mime != "" {
		return mime
	} else {
		return "text/plain"
	}
}

func setupRoutes(router *gin.Engine) {
	router.GET("/", API_Home)
	router.GET("/static/:type/:name", API_ServeAsset)

	api := router.Group("/api")
	{
		api.POST("/connect", API_Connect)
		api.GET("/databases", API_GetDatabases)
		api.GET("/connection", API_ConnectionInfo)
		api.GET("/tables", API_GetTables)
		api.GET("/tables/:table", API_GetTable)
		api.GET("/tables/:table/info", API_GetTableInfo)
		api.GET("/tables/:table/indexes", API_TableIndexes)
		api.GET("/query", API_RunQuery)
		api.POST("/query", API_RunQuery)
		api.GET("/explain", API_ExplainQuery)
		api.POST("/explain", API_ExplainQuery)
		api.GET("/history", API_History)
		api.GET("/bookmarks", API_Bookmarks)
	}

}

func API_Home(c *gin.Context) {
	data, err := Asset("static/index.html")

	if err != nil {
		c.String(400, err.Error())
		return
	}

	c.Data(200, "text/html; charset=utf-8", data)
}

func API_Connect(c *gin.Context) {
	url := c.Request.FormValue("url")

	if url == "" {
		c.JSON(400, Error{"Url parameter is required"})
		return
	}

	client, err := NewClientFromUrl(url)
	if err != nil {
		c.JSON(400, Error{err.Error()})
		return
	}

	err = client.Test()
	if err != nil {
		c.JSON(400, Error{err.Error()})
		return
	}

	info, err := client.Info()

	if err == nil {
		if dbClient != nil {
			dbClient.db.Close()
		}

		dbClient = client
	}

	c.JSON(200, info.Format()[0])
}

func API_GetDatabases(c *gin.Context) {
	names, err := dbClient.Databases()

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, names)
}

func API_RunQuery(c *gin.Context) {
	query := strings.TrimSpace(c.Request.FormValue("query"))

	if query == "" {
		c.JSON(400, errors.New("Query parameter is missing"))
		return
	}

	API_HandleQuery(query, c)
}

func API_ExplainQuery(c *gin.Context) {
	query := strings.TrimSpace(c.Request.FormValue("query"))

	if query == "" {
		c.JSON(400, errors.New("Query parameter is missing"))
		return
	}

	API_HandleQuery(fmt.Sprintf("EXPLAIN ANALYZE %s", query), c)
}

func API_GetTables(c *gin.Context) {
	names, err := dbClient.Tables()

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, names)
}

func API_GetTable(c *gin.Context) {
	res, err := dbClient.Table(c.Params.ByName("table"))

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, res)
}

func API_GetTableInfo(c *gin.Context) {
	res, err := dbClient.TableInfo(c.Params.ByName("table"))

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, res.Format()[0])
}

func API_History(c *gin.Context) {
	c.JSON(200, dbClient.history)
}

func API_ConnectionInfo(c *gin.Context) {
	if dbClient == nil {
		c.JSON(400, Error{"Not connected"})
		return
	}

	res, err := dbClient.Info()

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, res.Format()[0])
}

func API_TableIndexes(c *gin.Context) {
	res, err := dbClient.TableIndexes(c.Params.ByName("table"))

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, res)
}

func API_HandleQuery(query string, c *gin.Context) {
	result, err := dbClient.Query(query)

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	q := c.Request.URL.Query()

	if len(q["format"]) > 0 {
		if q["format"][0] == "csv" {
			c.Data(200, "text/csv", result.CSV())
			return
		}
	}

	c.JSON(200, result)
}

func API_Bookmarks(c *gin.Context) {
	bookmarks, err := readAllBookmarks()

	if err != nil {
		c.JSON(400, NewError(err))
		return
	}

	c.JSON(200, bookmarks)
}

func API_ServeAsset(c *gin.Context) {
	file := fmt.Sprintf(
		"static/%s/%s",
		c.Params.ByName("type"),
		c.Params.ByName("name"),
	)

	data, err := Asset(file)

	if err != nil {
		c.String(400, err.Error())
		return
	}

	if len(data) == 0 {
		c.String(404, "Asset is empty")
		return
	}

	c.Data(200, assetContentType(file), data)
}
