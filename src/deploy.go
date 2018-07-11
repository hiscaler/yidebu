package main

import (
	"os/exec"
	"fmt"
	"strings"
	"github.com/kr/pretty"
	"logger"
	"config"
	"path/filepath"
	ftp2 "github.com/jlaffaye/ftp"
	"os"
	"path"
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
	hostname string
	port     string
	username string
	password string
	rootPath string
}

func parseCommandReturnResult(s string) []string {
	res := make([]string, 0)
	rows := strings.Split(s, "\n")
	rows = rows[0 : len(rows)-1] // Remove command prompt line
	for _, row := range rows {
		row = strings.Trim(row, "\r\n\\\"")
		if len(row) != 0 {
			res = append(res, row)
		}
	}

	return res
}

func main() {
	// 获取项目配置
	cfg := config.Instance()
	cfgPathPrefix := "projects.Demo."
	project := new(Project)
	project.GitDir = filepath.ToSlash(cfg.GetString(cfgPathPrefix + "gitDir"))
	project.Dir = project.GitDir[:len(project.GitDir)-4]
	project.Ftp.hostname = cfg.GetString(cfgPathPrefix + "ftp.host")
	project.Ftp.port = cfg.GetString(cfgPathPrefix + "ftp.port")
	project.Ftp.username = cfg.GetString(cfgPathPrefix + "ftp.username")
	project.Ftp.password = cfg.GetString(cfgPathPrefix + "ftp.password")
	project.Ftp.rootPath = cfg.GetString(cfgPathPrefix + "ftp.rootPath")

	cmd := exec.Command("cmd", "/Y", "/Q", "/K", `git --git-dir=`+project.GitDir+` log --pretty=format:"%cn|%H|%cd|%s" -10`)
	out, err := cmd.Output()
	if err != nil {
		logger.Instance.Info(fmt.Sprintf("Run return erros: %s\n", err))
	} else {
		fmt.Printf("Raw output: \n%s\n", out)
		commits := make([]Commit, 0)
		rows := parseCommandReturnResult(string(out))
		if len(rows) > 0 {
			for _, row := range rows {
				c := strings.Trim(string(row), "\"\r\n")
				fmt.Println("row = " + c)
				t := strings.Split(string(c), "|")
				commits = append(commits, Commit{
					t[0], t[1], t[2], t[3],
				})
			}
			fmt.Println(fmt.Sprintf("%# v", pretty.Formatter(commits)))
			updateFiles := make(map[string]string, 0)
			for _, commit := range commits {
				fmt.Println(fmt.Sprintf("%# v", pretty.Formatter(commit)))
				gitShowCommand := `git --git-dir=` + project.GitDir + ` show ` + commit.id + ` --name-only --pretty=format:"%f"`
				cmd = exec.Command("cmd", "/Y", "/Q", "/K", gitShowCommand)
				out, err = cmd.Output()
				if err != nil {
					logger.Instance.Error(fmt.Sprintf("Run return erros: %s\n", err))
				} else {
					rows = parseCommandReturnResult(string(out))
					for _, row := range rows {
						updateFiles[row] = row
					}
					logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(rows)))
				}
			}

			if len(updateFiles) > 0 {
				fmt.Println("Update files")
				ftpClient, err := ftp2.Connect(project.Ftp.hostname + ":" + project.Ftp.port)
				if err == nil {
					if err := ftpClient.Login(project.Ftp.username, project.Ftp.password); err != nil {
						logger.Instance.Error("FTP login error: " + err.Error())
					} else {
						ftpClient.ChangeDir(project.Ftp.rootPath)
						for _, file := range updateFiles {
							dirs := strings.Split(file, "/")
							for _, dir := range dirs {
								if err := ftpClient.ChangeDir(path.Join(project.Ftp.rootPath, dir)); err != nil {
									if err := ftpClient.MakeDir(path.Join(project.Ftp.rootPath, dir)); err != nil {
										ftpClient.ChangeDir(path.Join(project.Ftp.rootPath, dir))
									} else {
										logger.Instance.Error("FTP make dir error: " + err.Error())
										panic("FTP make dir error: " + err.Error())
									}
								}
							}
							logger.Instance.Info(fmt.Sprintf("%s", file))
							// Use FTP upload file
							fullPath := filepath.Join(project.Dir, file)
							fmt.Println(fullPath)
							if f, err := os.Open(fullPath); err != nil {
								ftpClient.Stor(filepath.Join(project.Ftp.rootPath, file), f)
							} else {
								logger.Instance.Error("FTP Store file error: " + err.Error())
							}
						}
					}
				} else {
					logger.Instance.Error("FTP connection error: " + err.Error())
				}
			} else {
				logger.Instance.Info("No update files.")
			}
		} else {
			logger.Instance.Error("Not find commits")
		}
	}
}
