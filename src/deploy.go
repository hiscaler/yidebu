package main

import (
	"os/exec"
	"fmt"
	"strings"
	"github.com/kr/pretty"
	"logger"
	"config"
	"path/filepath"
	"os"
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
	rows = rows[0 : len(rows)-1] // Remove command prompt line
	for _, row := range rows {
		row = strings.Trim(row, "\r\n\\\"")
		if len(row) != 0 {
			res = append(res, row)
		}
	}

	return res
}

// 基于 git 的简易代码 FTP 部署工具
func main() {
	// 获取项目配置
	cfg := config.Instance()
	cfgPathPrefix := "projects.demo"
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
	
	cmd := exec.Command("cmd", "/Y", "/Q", "/K", `git --git-dir=`+activeProject.GitDir+` log --pretty=format:"%cn|%H|%cd|%s" -10`)
	out, err := cmd.Output()
	if err != nil {
		logger.Instance.Info(fmt.Sprintf("Run return erros: %s\n", err))
	} else {
		logger.Instance.Info(fmt.Sprintf("Raw content: %s", out))
		commits := make([]Commit, 0)
		rows := parseCommandReturnResult(string(out))
		if len(rows) > 0 {
			for _, row := range rows {
				c := strings.Trim(string(row), "\"\r\n")
				logger.Instance.Info("row = " + c)
				t := strings.Split(string(c), "|")
				commits = append(commits, Commit{
					t[0], t[1], t[2], t[3],
				})
			}
			logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(commits)))
			updateFiles := make(map[string]string, 0)
			for _, commit := range commits {
				fmt.Println(fmt.Sprintf("%# v", pretty.Formatter(commit)))
				gitShowCommand := `git --git-dir=` + activeProject.GitDir + ` show ` + commit.id + ` --name-only --pretty=format:"%f"`
				cmd = exec.Command("cmd", "/Y", "/Q", "/K", gitShowCommand)
				out, err = cmd.Output()
				if err != nil {
					logger.Instance.Error(fmt.Sprintf("Run return erros: %s\n", err))
				} else {
					rows = parseCommandReturnResult(string(out))
					for _, row := range rows {
						updateFiles[row] = row
					}
					logger.Instance.Info(fmt.Sprintf("%# v", pretty.Formatter(updateFiles)))
				}
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
							defer f.Close()
							if err == nil {
								if err := ftpClient.Stor(path.Base(file), f); err == nil {
									uploadedFilesCount += 1
									logger.Instance.Info("FTP store file success: " + path.Clean(file))
								} else {
									logger.Instance.Info("FTP store file error: " + err.Error())
								}
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
		} else {
			logger.Instance.Error("Not find commits")
		}
	}
}
