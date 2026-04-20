package handler

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// parseIDFromVars 从路由变量中读取整数 ID。
func parseIDFromVars(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
}

// parsePage 读取分页参数。
func parsePage(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
