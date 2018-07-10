package main

import (
	"os/exec"
	"fmt"
	"strings"
	"github.com/kr/pretty"
	"logger"
	"config"
	"path/filepath"
)

type Commit struct {
	author   string
	id       string
	datetime string
	comment  string
}

type Project struct {
	gitDir     string
	ignoreFile []string
	ftp        FTP
}

type FTP struct {
	hostname string
	port     int
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
	projectName := "demo"
	project := Project{}
	cfg.Configure(&project, "projects."+projectName)
	cmd := exec.Command("cmd", "/Y", "/Q", "/K", `git --git-dir=`+project.gitDir+` log --pretty=format:"%cn|%H|%cd|%s" -10`)
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
				gitShowCommand := `git --git-dir=` + project.gitDir + ` show ` + commit.id + ` --name-only --pretty=format:"%f"`
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
				projectDir := project.gitDir[:len(project.gitDir)-4]
				for _, file := range updateFiles {
					logger.Instance.Info(fmt.Sprintf("%s", file))
					// Use FTP upload file
					fullPath := filepath.Join(projectDir, file)
					fmt.Println(fullPath)
				}
			} else {
				logger.Instance.Info("No update files.")
			}
		} else {
			logger.Instance.Error("Not find commits")
		}
	}
}
