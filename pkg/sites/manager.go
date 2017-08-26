package sites

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	logger "github.com/Sirupsen/logrus"
	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/acme/autocert"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
)

type storageContents struct {
	StaticSites  map[string]*StaticSite `json:"static_dirs"`
	WildCardHost string                 `json:"wild_card"`
}
type apiRespContents struct {
	StaticSites  map[string]*StaticSitePublic `json:"static_dirs"`
	WildCardHost string                       `json:"wild_card"`
}

func writeHTTPError(w http.ResponseWriter, err error) {
	writer := json.NewEncoder(w)
	writer.Encode(&struct {
		Error string `json:"error"`
	}{err.Error()})
}

func hashSha(password string) string {
	s := sha1.New()
	s.Write([]byte(password))
	passwordSum := []byte(s.Sum(nil))
	return base64.StdEncoding.EncodeToString(passwordSum)
}

// Server
type manager struct {
	sync.RWMutex
	staticSites   map[string]*StaticSite
	dataDir       string
	wildCardHost  string
	router        http.Handler
	staticFs      http.FileSystem
	adminUsername string
	adminPassword string
}

func (s *manager) dataPath() string {
	return path.Join(s.dataDir, "data.json")
}

func (s *manager) repoDir(id string) string {
	return path.Join(s.dataDir, "repos", id)
}

func (s *manager) certDir() string {
	return path.Join(s.dataDir, "certs")
}

func (s *manager) init(cloneOnInit bool) error {
	err := os.MkdirAll(s.dataDir, 0700)
	if err != nil {
		return err
	}
	if _, err := os.Stat(s.dataPath()); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	data, err := ioutil.ReadFile(s.dataPath())
	if err != nil {
		return err
	}
	contents := &storageContents{}
	err = json.Unmarshal(data, contents)
	if err != nil {
		return err
	}
	for _, site := range contents.StaticSites {
		err := s.AddStaticSite(site, cloneOnInit, false)
		if err != nil {
			logger.Errorf("Error initializing repo %s: %s", site.GitURL, err)
			return err
		}
	}
	return s.save()
}

func (s *manager) AddStaticSite(site *StaticSite, clone bool, save bool) error {
	defer s.updateRoutes()
	blockedHosts := []string{s.wildCardHost}
	for _, site := range s.staticSites {
		blockedHosts = append(blockedHosts, site.Hostnames...)
	}
	err := site.Verify(blockedHosts)
	site.cloneTo = s.repoDir(site.ID)
	if err != nil {
		return err
	}
	if clone {
		err = site.Update()
		if err != nil {
			logger.Error(err)
			return errors.New("Unable to clone repo.")
		}
	}
	s.Lock()
	defer s.Unlock()
	s.staticSites[site.ID] = site
	if save {
		err = s.save()
		if err != nil {
			return err
		}
	}
	site.Start()
	return nil
}

func (s *manager) DeleteStaticSite(id string, password string, save bool) error {
	defer s.updateRoutes()
	s.Lock()
	defer s.Unlock()
	site, ok := s.staticSites[id]
	if ok {
		if site.DeletePassword != password {
			return errors.New("Password does not match.")
		}
		site.Stop()
		defer os.RemoveAll(s.repoDir(id))
	}
	delete(s.staticSites, id)
	if save {
		err := s.save()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *manager) save() error {
	toStore := &storageContents{StaticSites: s.staticSites}
	data, err := json.Marshal(toStore)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.dataPath(), data, 0700)
	return err
}

func (s *manager) Save() error {
	s.Lock()
	defer s.Unlock()
	err := s.save()
	return err
}

func (s *manager) addSiteHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	site := &StaticSite{}

	if err := decoder.Decode(site); err != nil {
		writeHTTPError(w, err)
		return
	}

	if err := s.AddStaticSite(site, true, true); err != nil {
		writeHTTPError(w, err)
		return
	}
	w.Write([]byte("{}"))
}

func (s *manager) listSiteHandler(w http.ResponseWriter, r *http.Request) {
	staticSites := map[string]*StaticSitePublic{}
	for id, site := range s.staticSites {
		staticSites[id] = site.PublicRepr()
	}
	toStore := &apiRespContents{StaticSites: staticSites, WildCardHost: s.wildCardHost}
	encoder := json.NewEncoder(w)
	encoder.Encode(toStore)
}

func (s *manager) deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	password := vars["deletePass"]
	err := s.DeleteStaticSite(id, password, true)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	w.Write([]byte("{}"))
}

func (s *manager) updateRoutes() {
	r := mux.NewRouter().StrictSlash(false)
	s.RLock()
	defer s.RUnlock()

	// Create static site routes
	for _, site := range s.staticSites {
		dataPath := site.cloneTo
		if site.Root != "" || site.Root != "/" {
			dataPath = path.Join(site.cloneTo, site.Root)
		}
		for _, host := range site.Hostnames {
			hostHandler := http.FileServer(http.Dir(dataPath))
			// Authenticate static sites with auth settings
			if site.AccessUsername != "" && site.AccessPassword != "" {
				subPathAuth := auth.NewBasicAuthenticator(host, func(user, realm string) string {
					if user == site.AccessUsername {
						return "{SHA}" + hashSha(site.AccessPassword)
					}
					return ""
				})
				hostHandler = auth.JustCheck(subPathAuth, hostHandler.ServeHTTP)
			}
			r.HandleFunc("/.git/{rest:.*}", http.NotFound).Host(host)
			r.Handle("/{rest:.*}", hostHandler).Host(host)
		}
	}

	// The API
	apiRouter := mux.NewRouter().StrictSlash(false)
	apiRouter.HandleFunc("/v1/sites", s.addSiteHandler).Methods("POST")
	apiRouter.HandleFunc("/v1/sites", s.listSiteHandler).Methods("GET")
	apiRouter.HandleFunc("/v1/sites/{id}", s.deleteHandler).Methods("DELETE")
	apiRouter.HandleFunc("/v1/sites/{id}/{deletePass}", s.deleteHandler).Methods("DELETE")
	var apiHandler http.HandlerFunc
	if s.adminPassword != "" {
		apiAuth := auth.NewBasicAuthenticator("api", func(user, realm string) string {
			if user == s.adminUsername {
				return "{SHA}" + hashSha(s.adminPassword)
			}
			return ""
		})
		apiHandler = auth.JustCheck(apiAuth, apiRouter.ServeHTTP)
	} else {
		apiHandler = apiRouter.ServeHTTP
	}

	r.Handle("/v1/{rest:.*}", apiHandler)

	if s.staticFs != nil {
		r.Handle("/{rest:.*}", http.FileServer(s.staticFs))
	}
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	s.router = loggedRouter
}

func (s *manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *manager) RunHTTPS(listenAddr string) error {
	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(s.certDir()),
	}
	server := &http.Server{
		Addr:      listenAddr,
		Handler:   s,
		TLSConfig: &tls.Config{GetCertificate: certManager.GetCertificate},
	}
	return server.ListenAndServeTLS("", "")
}

func (s *manager) RunHTTP(listenAddr string) error {
	server := &http.Server{
		Addr:    listenAddr,
		Handler: s,
	}
	return server.ListenAndServe()
}

func NewManager(dataDir string, wildCardHost string, fs http.FileSystem, adminUsername string, adminPassword string) (*manager, error) {
	dataDir, err := homedir.Expand(dataDir)
	if err != nil {
		return nil, err
	}
	manager := &manager{
		dataDir:       dataDir,
		staticSites:   map[string]*StaticSite{},
		router:        mux.NewRouter(),
		wildCardHost:  wildCardHost,
		staticFs:      fs,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}
	// Don't clone on init so a bad repo doesn't block restarting
	err = manager.init(false)
	if err != nil {
		return nil, err
	}
	manager.updateRoutes()
	return manager, nil
}
