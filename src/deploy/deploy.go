package main

import (
	"fmt"
	"strings"
	"logger"
	"config"
	"flag"
	"path/filepath"
	"os"
	"os/exec"
	"github.com/kr/pretty"
	"strconv"
	"path"
	"github.com/jlaffaye/ftp"
)

type Commit struct {
	author   string
	id       string
	datetime string
	comment  string
}

type Project struct {
	GitDir      string
	Dir         string
	IgnoreFiles []string
	Ftp         FTP
}

type FTP struct {
	Hostname string
	Port     string
	Username string
	Password string
	RootPath string
}

func parseCommandReturnResult(s string) []string {
	res := make([]string, 0)
	rows := strings.Split(s, "\n")
	if len(rows) > 0 {
		rows = rows[0 : len(rows)-1] // Remove command prompt line
	}
	for _, row := range rows {
		row = strings.Trim(row, "\r\n\\\"")
		if len(row) != 0 {
			res = append(res, row)
		}
	}

	return res
}

type Git struct {
	name              string
	path              string
	branch            string // 当前所操作的分支
	tag               string //  当前所操作的标签
	fetchCommitNumber int    // 拉取的提交数量
	project           Project
}

func (g *Git) execCommand(args ...string) ([]byte, error) {
	gitCommand := `git --git-dir=` + g.path
	if len(args) > 0 {
		gitCommand += " " + strings.Join(args[:], " ")
	}
	logger.Instance.Info(gitCommand)
	cmd := exec.Command("cmd", "/Y", "/Q", "/K", gitCommand)

	return cmd.Output()
}

func (g *Git) changeBranch() (bool, error) {
	if g.branch == "" {
		g.branch = "master"
	}
	_, err := g.execCommand("checkout", g.branch)

	return err == nil, err
}

func (g *Git) commits() []Commit {
	commits := make([]Commit, 0)
	g.changeBranch()
	rows := make([]string, 0)
	if g.tag != "" {
		if g.hasTag() {
			// Get by tag
			sign := "git-tag-sign:"
			out, err := g.execCommand(`show `+g.tag+` --format="`+sign+`%cn|%H|%cd|%s"`, "-n"+strconv.Itoa(g.fetchCommitNumber))
			if err != nil {
				logger.Instance.Info(fmt.Sprintf("Run return erros: %s\n", err))
			} else {
				logger.Instance.Info(fmt.Sprintf("Raw content: %s", out))
				t := strings.Split(string(out), "\n")
				for _, row := range t {
					row = strings.Trim(row, "\r\n\\\"")
					if len(row) != 0 && strings.HasPrefix(row, sign) {
						rows = append(rows, row[len(sign):])
					}
				}
			}
		}
	} else {
		// Get by latest commits
		out, err := g.execCommand(`log --pretty=format:"%cn|%H|%cd|%s`, "-"+strconv.Itoa(g.fetchCommitNumber))
		if err != nil {
			logger.Instance.Info(fmt.Sprintf("Run return erros: %s\n", err))
		} else {
			logger.Instance.Info(fmt.Sprintf("Raw content: %s", out))
			rows := parseCommandReturnResult(string(out))
			fmt.Println(rows)
		}
	}

	if len(rows) > 0 {
		for _, row := range rows {
			c := strings.Trim(string(row), "\"\r\n")
			logger.Instance.Info("row = " + c)
			t := strings.Split(string(c), "|")
			commits = append(commits, Commit{
				t[0], t[1], t[2], t[3],
			})
		}
	}
	logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(commits)))

	return commits
}

// 显示当前 git 仓库的所有标签
func (g *Git) tags() []string {
	res, err := g.execCommand("tag")
	if err == nil {
		return parseCommandReturnResult(string(res))
	} else {
		return make([]string, 0)
	}
}

func (g *Git) hasTag() bool {
	has := false
	if len(g.tag) > 0 {
		for _, tag := range g.tags() {
			if tag == g.tag {
				has = true
				break
			}
		}
	}

	return has
}

// Get update and delete files
func (g *Git) Files() ([]string, []string) {
	updateFiles := make([]string, 0)
	deleteFiles := make([]string, 0)
	commits := g.commits()
	logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(commits)))
	for _, commit := range commits {
		fmt.Println(fmt.Sprintf("%# v", pretty.Formatter(commit)))
		out, err := g.execCommand("show", commit.id, `--name-only --pretty=format:"%f"`)
		if err != nil {
			logger.Instance.Error(fmt.Sprintf("Run return erros: %s\n", err))
		} else {
			rows := parseCommandReturnResult(string(out))
			for _, row := range rows {
				ignore := false
				for _, f := range g.project.IgnoreFiles {
					if f == row {
						ignore = true
					}
				}
				if !ignore {
					// unique file name in `updateFiles` variable
					exists := false
					for _, t := range updateFiles {
						if row == t {
							exists = true
							break
						}
					}
					if !exists {
						updateFiles = append(updateFiles, row)
					}
				}
			}
			logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(updateFiles)))
		}
	}

	return updateFiles, deleteFiles
}

// 基于 git 的简易代码 FTP 部署工具
func main() {
	var (
		p          string // Project name
		branchName string // Git branch name
		tag        string // Git tag
		n          int    // How much commits per time
		h          bool   // Helps
	)
	flag.BoolVar(&h, "h", false, "使用说明")
	flag.StringVar(&p, "p", "", "处理的项目名称")
	flag.IntVar(&n, "n", 20, "要拉取的数据条数")
	flag.StringVar(&branchName, "b", "master", "分支名称")
	flag.StringVar(&tag, "t", "", "标签名")
	flag.Parse()
	if h {
		fmt.Fprintf(os.Stderr, `使用方法: ./deploy [-hpnbt] [-p project-name] [-n fetch numbers] [-b branch-name] [-t tag-name]
参数详细说明:
`)
		flag.PrintDefaults()
		os.Exit(0)
	}

	if p == "" {
		p = "demo"
	}
	if n == 0 {
		n = 20
	}
	if branchName == "" {
		branchName = "master"
	}

	fmt.Printf("Project = %s, n = %d\n", p, n)

	// 获取项目配置
	cfg := config.Instance()
	projects := make(map[string]Project, 0)
	cfg.Configure(&projects, "projects")
	exists := false
	if len(p) > 0 {
		for name, _ := range projects {
			if p == name {
				exists = true
				break
			}
		}
	}
	if exists {
		name := p
		cfgPathPrefix := "projects." + name
		logger.Instance.Info("Project `" + name + "` Begin...")
		activeProject := new(Project)
		cfg.Configure(&activeProject, cfgPathPrefix)
		logger.Instance.Info(fmt.Sprintf("%#v", activeProject))
		activeProject.GitDir = filepath.ToSlash(activeProject.GitDir)
		if len(activeProject.GitDir) == 0 {
			logger.Instance.Error("请检查配置文件是否正确。")
			os.Exit(0)
		}
		if len(activeProject.GitDir) > 4 {
			activeProject.Dir = activeProject.GitDir[:len(activeProject.GitDir)-4]
		}

		git := new(Git)
		git.name = name
		git.path = activeProject.GitDir
		git.branch = branchName
		git.tag = tag
		git.project = *activeProject
		git.fetchCommitNumber = n
		fmt.Println(fmt.Sprintf("#%v", git))

		updateFiles, _ := git.Files()
		logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(updateFiles)))
		for _, file := range updateFiles {
			fmt.Println(file)
		}

		if len(updateFiles) > 0 {
			fmt.Println("Update files")
			ftpClient, err := ftp.Connect(activeProject.Ftp.Hostname + ":" + activeProject.Ftp.Port)
			if err == nil {
				defer ftpClient.Quit()
				if err := ftpClient.Login(activeProject.Ftp.Username, activeProject.Ftp.Password); err != nil {
					logger.Instance.Error("FTP login error: " + err.Error())
				} else {
					uploadedFilesCount := 0
					for _, file := range updateFiles {
						file = filepath.ToSlash(file)
						ftpClient.ChangeDir(activeProject.Ftp.RootPath)
						tempPath := activeProject.Ftp.RootPath
						dirs := strings.Split(path.Dir(file), "/")
						for _, dir := range dirs {
							currentDir, _ := ftpClient.CurrentDir()
							ftpClient.ChangeDir(currentDir)
							tempPath = path.Join(tempPath, dir)
							if err := ftpClient.ChangeDir(tempPath); err != nil {
								if err := ftpClient.MakeDir(tempPath); err == nil {
									ftpClient.ChangeDir(tempPath)
								} else {
									logger.Instance.Error("FTP make dir error: " + err.Error())
									panic("FTP make dir error: " + err.Error())
								}
							}
						}
						logger.Instance.Info(fmt.Sprintf("%s", file))
						// Use FTP upload file
						localFilePath := filepath.Join(activeProject.Dir, file)
						logger.Instance.Info("Local file: " + localFilePath)
						logger.Instance.Info("Remote file: " + file)
						f, err := os.Open(localFilePath)
						if err == nil {
							if err := ftpClient.Stor(path.Base(file), f); err == nil {
								uploadedFilesCount += 1
								logger.Instance.Info("FTP store file success: " + path.Clean(file))
							} else {
								logger.Instance.Info("FTP store file error: " + err.Error())
							}
							f.Close()
						} else {
							logger.Instance.Info("Open file error: " + err.Error())
						}
					}

					logger.Instance.Info(fmt.Sprintf("Total %d files, %d files success upploaded", len(updateFiles), uploadedFilesCount))
				}
			} else {
				logger.Instance.Error("FTP connection error: " + err.Error())
			}
		} else {
			logger.Instance.Info("No update files.")
		}

		logger.Instance.Info("Project `" + name + "` Done...")
	} else {
		logger.Instance.Error(p + " Project not exists.")
	}

}
