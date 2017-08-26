package sites

import (
	"errors"
	"fmt"
	logger "github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/ssh"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"strings"
	"sync"
	"time"
)

const repoSyncInterval = time.Second * 15

type StaticSite struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Hostnames      []string `json:"hostnames"`
	GitURL         string   `json:"git_repo"`
	Branch         string   `json:"branch"`
	Root           string   `json:"root"`
	DeletePassword string   `json:"delete_password"`
	AccessUsername string   `json:"access_username"`
	AccessPassword string   `json:"access_password"`
	SshKey         string   `json:"ssh_key"`
	healthy        bool
	cloneTo        string
	repo           *git.Repository
	repoWT         *git.Worktree
	gitAuth        transport.AuthMethod
	closeChan      chan bool
	sync.Mutex
}

// Exposes public fields of a static site for api response
type StaticSitePublic struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Hostnames         []string `json:"hostnames"`
	GitURL            string   `json:"git_repo"`
	Branch            string   `json:"branch"`
	Root              string   `json:"root"`
	Healthy           bool     `json:"healthy"`
	HasDeletePassword bool     `json:"has_delete_password"`
}

func (sd *StaticSite) CloseChan() chan bool {
	if sd.closeChan == nil {
		sd.closeChan = make(chan bool)
	}
	return sd.closeChan

}

func (sd *StaticSite) PublicRepr() *StaticSitePublic {
	hasDeletePassword := len(sd.DeletePassword) > 1
	return &StaticSitePublic{
		ID:                sd.ID,
		Name:              sd.Name,
		Hostnames:         sd.Hostnames,
		GitURL:            sd.GitURL,
		Branch:            sd.Branch,
		Root:              sd.Root,
		Healthy:           sd.healthy,
		HasDeletePassword: hasDeletePassword,
	}
}

func (sd *StaticSite) Verify(protectedHostnames []string) error {
	if sd.Name == "" {
		return errors.New("Please provide a name.")
	}
	if len(sd.Hostnames) == 0 {
		return errors.New("Hostnames are required.")
	}
	if sd.Branch == "" {
		sd.Branch = "master"
	}
	if sd.Root == "" {
		sd.Root = "/"
	}
	if (sd.AccessUsername != "" && sd.AccessPassword == "") || (sd.AccessUsername == "" && sd.AccessPassword != "") {
		return errors.New("Both username and password must be provided")
	}

	if sd.AccessPassword != "" && len(sd.AccessPassword) < 5 {
		return errors.New("Password must be 4 or more characters long.")
	}
	if sd.ID == "" {
		sd.ID = uuid.NewV4().String()
	}
	if sd.SshKey != "" {
		_, err := ssh.ParsePrivateKey([]byte(sd.SshKey))
		if err != nil {
			return errors.New("Unable to parse ssh key. Note: the key should not have a passphrase.")
		}
	}
	hostnames := []string{}
	for _, hostname := range sd.Hostnames {
		for _, protected := range protectedHostnames {
			if hostname == protected {
				return errors.New(fmt.Sprintf("Cannot use hostname %s", hostname))
			}
		}
		if hostname != "" {
			hostnames = append(hostnames, strings.ToLower(hostname))
		}
	}
	sd.Hostnames = hostnames
	return nil
}

func (sd *StaticSite) openOrClone() error {
	if sd.SshKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(sd.SshKey))
		if err != nil {
			return err
		}
		sd.gitAuth = &gitssh.PublicKeys{User: "git", Signer: signer}
	}
	repo, err := git.PlainOpen(sd.cloneTo)
	if err == git.ErrRepositoryNotExists {
		logger.Infof("Started initial clone of repo %s", sd)
		repo, err = git.PlainClone(sd.cloneTo, false, &git.CloneOptions{
			Auth:          sd.gitAuth,
			URL:           sd.GitURL,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", sd.Branch)),
		})
		logger.Infof("Completed initial clone of repo %s", sd)
	}
	if err != nil {
		return err
	}

	sd.repo = repo
	wt, err := sd.repo.Worktree()
	if err != nil {
		return nil
	}
	sd.repoWT = wt
	return nil
}

func (sd *StaticSite) GitRef() plumbing.ReferenceName {
	return plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", sd.Branch))
}

func (sd *StaticSite) update() error {
	sd.Lock()
	defer sd.Unlock()
	if sd.repo == nil {
		err := sd.openOrClone()
		if err != nil {
			return err
		}
	}
	err := sd.repo.Fetch(&git.FetchOptions{
		Auth: sd.gitAuth,
	})
	if err != git.NoErrAlreadyUpToDate {
		return err
	}

	currentHead, err := sd.repo.Reference(plumbing.HEAD, true)
	if err != nil {
		return err
	}

	targetRef, err := sd.repo.Reference(sd.GitRef(), true)
	if err != nil {
		return err
	}

	err = sd.repoWT.Reset(&git.ResetOptions{
		Commit: targetRef.Hash(),
	})
	if err != nil {
		return err
	}

	if currentHead.Hash() != targetRef.Hash() {
		logger.Infof("Updated repo %s from %s to %s", sd, currentHead.Hash(), targetRef.Hash())
	}

	return nil
}

func (sd *StaticSite) Update() error {
	logger.Debugf("Updating repo %s", sd)
	err := sd.update()
	if err != nil {
		sd.healthy = false
		logger.Warnf("Unable to update repo %s, error: %s", sd, err)
	} else {
		sd.healthy = true
		logger.Debugf("Updated repo %s", sd)
	}
	return err
}

func (sd *StaticSite) Start() {
	go func() {
		closeChan := sd.CloseChan()
		for {
			select {
			case <-time.After(repoSyncInterval):
				sd.Update()
			case <-closeChan:
				logger.Infof("Stopped watch on repo %s", sd)
				return
			}
		}
	}()
}

func (sd *StaticSite) String() string {
	return fmt.Sprintf("<Repo remote=%s branch=%s>", sd.GitURL, sd.Branch)
}

func (sd *StaticSite) Stop() {
	logger.Infof("Stopping watch on repo %s", sd)
	closeChan := sd.CloseChan()
	closeChan <- true
	sd.closeChan = nil
	return
}
