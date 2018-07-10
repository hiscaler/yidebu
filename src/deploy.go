package main

import (
	"os/exec"
	"fmt"
	"strings"
	"github.com/kr/pretty"
)

// git 项目路径
var gitDir = ""
// 忽略的文件
var ignoreFiles = make([]string, 0)

type Commit struct {
	author   string
	id       string
	datetime string
	comment  string
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
	cmd := exec.Command("cmd", "/Y", "/Q", "/K", `git --git-dir=`+gitDir+` log --pretty=format:"%cn|%H|%cd|%s" -10`)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Run return erros: %s\n", err)
	} else {
		fmt.Printf("Raw output: \n%s\n", out)
		commits := make([]Commit, 0)
		rows := strings.Split(string(out), "\n")
		rows = rows[0 : len(rows)-1] // Remove command prompt line
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
			gitShowCommand := `git --git-dir=` + gitDir + ` show ` + commit.id + ` --name-only --pretty=format:"%f"`
			cmd = exec.Command("cmd", "/Y", "/Q", "/K", gitShowCommand)
			out, err = cmd.Output()
			if err != nil {
				fmt.Printf("Run return erros: %s\n", err)
			} else {
				rows = parseCommandReturnResult(string(out))
				for _, row := range rows {
					updateFiles[row] = row
				}
				fmt.Println(fmt.Sprintf("%# v", pretty.Formatter(rows)))
			}
		}

		if len(updateFiles) > 0 {
			fmt.Println("Update files")
			fmt.Println(strings.Repeat("#", 80))
			for _, file := range updateFiles {
				fmt.Println(fmt.Sprintf("%s", file))
				// FTP upload file
				//fullPath := gitDir + file
				//fmt.Println(fullPath)
			}
		} else {
			fmt.Println("No update files.")
		}
	}
}
