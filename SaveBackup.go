package main

import (
	"bytes"
	"log"
	"strings"
	"text/template"

	"github.com/tailscale/walk"
)

const REG_FILE_HEADER = "Windows Registry Editor Version 5.00\n\n"

var packageInfo = template.New("packageInfo")
var tmplProperty = template.Must(packageInfo.Parse(string(`[{{.RegPath}}\Interrupt Management]

[{{.RegPath}}\Interrupt Management\Affinity Policy]
"DevicePolicy"=dword:{{printf "%08d" .Device.DevicePolicy}}
{{if eq .Device.DevicePriority 0}}"DevicePriority"=-{{else}}"DevicePriority"=dword:{{printf "%08d" .Device.DevicePriority}}{{end}}
{{if ne .Device.DevicePolicy 4}}"AssignmentSetOverride"=-{{else}}"AssignmentSetOverride"=hex:{{.AssignmentSetOverride}}{{end}}

[{{.RegPath}}\Interrupt Management\MessageSignaledInterruptProperties]
"MSISupported"=dword:{{printf "%08d" .Device.MsiSupported}}
{{if eq .Device.MsiSupported 1}}{{if ne .Device.MessageNumberLimit 0}}"MessageNumberLimit"=dword:{{printf "%08d" .Device.MessageNumberLimit}}{{end}}{{else}}"MessageNumberLimit"=-{{end}}

`)))

func createRegFile(dlg *walk.Dialog, regpath string, item Device) string {
	var buf bytes.Buffer
	err := tmplProperty.Execute(&buf, struct {
		RegPath               string
		Device                Device
		AssignmentSetOverride string
	}{
		regpath,
		item,
		addComma(ToLittleEndian(uint64(item.AssignmentSetOverride))),
	})

	if err != nil {
		walk.MsgBox(dlg, "CreateRegFile Error", err.Error(), walk.MsgBoxIconError)
		log.Fatalln(err)
	}

	return strings.ReplaceAll(buf.String(), "\n", "\r\n")

}

func addComma(data string) string {
	var b strings.Builder
	for i := 0; i < len(data); i++ {
		if i != 0 && i%2 == 0 {
			b.WriteString(",")
		}
		b.WriteString(string(data[i]))
	}

	return b.String()
}

func saveFileExplorer(owner walk.Form, path, filename, title, filter string) (filePath string, cancel bool, err error) {
	dlg := new(walk.FileDialog)

	dlg.Title = title
	dlg.InitialDirPath = path
	dlg.Filter = filter
	dlg.FilePath = filename

	ok, err := dlg.ShowSave(owner)
	if err != nil {
		return "", !ok, err
	} else if !ok {
		return "", !ok, nil
	}

	return dlg.FilePath, !ok, nil
}
