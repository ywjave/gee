package gee

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

type HandlerFunc func(*Context)

type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	engine      *Engine
}

type Engine struct {
	*RouterGroup
	router        *router
	groups        []*RouterGroup //记录所有的分组
	htmlTemplates *template.Template
	funcMap       template.FuncMap
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) LoadHTMLGlob(pattern string) {
	e.htmlTemplates = template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
}

func New() *Engine {
	engine := &Engine{router: NewRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engin := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		engine: engin,
	}
	engin.groups = append(engin.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method, comp string, hf HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, hf)
}

func (group *RouterGroup) GET(pattern string, hf HandlerFunc) {
	group.addRoute("GET", pattern, hf)
}

func (group *RouterGroup) POST(pattern string, hf HandlerFunc) {
	group.addRoute("POST", pattern, hf)
}

func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range e.groups {
		if strings.HasPrefix(r.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, r)
	c.handlers = middlewares
	c.e = e
	e.router.handle(c)
}

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileserver := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileserver.ServeHTTP(c.Writer, c.Req)
	}
}

func (group *RouterGroup) Satic(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	group.GET(urlPattern, handler)
}

func (e *Engine) Run(addr string) (err error) {
	fmt.Println("start run")
	return http.ListenAndServe(addr, e)
}
