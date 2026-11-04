package desktop

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"plainmark/internal/cryptoblock"
	"plainmark/internal/session"
)

type cryptoForm struct {
	kind                                           string
	doc                                            *Document
	from, to                                       int
	version                                        uint64
	plaintext, payload, password, confirm, message string
}

func (a *App) beginEncrypt() {
	d := a.current()
	if d == nil || a.busy {
		return
	}
	if d.Tab.Mode == session.Preview {
		a.status = "Switch to Edit or Split and select text first"
		return
	}
	sels := d.Code.Selections()
	if len(sels) == 0 || sels[0].Empty() {
		a.status = "Select text to encrypt"
		return
	}
	s := sels[0]
	a.crypto = &cryptoForm{kind: "encrypt", doc: d, from: s.Start(), to: s.End(), version: d.Code.Buffer.Version(), plaintext: d.Code.Buffer.Slice(s.Start(), s.End())}
}
func (a *App) submitCrypto() {
	if a.crypto == nil || a.busy {
		return
	}
	f := *a.crypto
	if f.password == "" {
		a.crypto.message = "Password cannot be empty"
		return
	}
	if f.kind == "encrypt" && f.password != f.confirm {
		a.crypto.message = "Passwords do not match"
		return
	}
	a.async(func() (func(), error) {
		var result string
		var err error
		if f.kind == "encrypt" {
			result, err = cryptoblock.Encrypt(f.plaintext, f.password)
		} else {
			result, err = cryptoblock.Decrypt(f.payload, f.password)
		}
		if err != nil {
			return func() {
				if a.crypto != nil {
					a.crypto.message = err.Error()
				}
			}, nil
		}
		return func() {
			if a.docs[f.doc.Tab] != f.doc {
				a.closeCrypto()
				return
			}
			if f.kind == "encrypt" {
				buf := f.doc.Code.Buffer
				if buf.Version() != f.version || buf.Slice(f.from, f.to) != f.plaintext {
					a.crypto.message = "Selection changed. Select it again."
					return
				}
				prefix, suffix := "", ""
				if f.from > 0 && buf.Slice(f.from-1, f.from) != "\n" {
					prefix = "\n\n"
				}
				if f.to < buf.Len() && buf.Slice(f.to, f.to+1) != "\n" {
					suffix = "\n\n"
				}
				block := prefix + "```encrypted\n" + strings.TrimSpace(result) + "\n```" + suffix
				buf.Transact(func() { buf.Edit(editor.Edit{Start: f.from, End: f.to, Text: block}) })
				f.doc.Code.Select(f.from+len(block), f.from+len(block))
				f.doc.Tab.Draft = buf.String()
				a.status = "Selection encrypted; save the note to write it to disk"
			} else {
				f.doc.Unlocked[f.payload] = result
				f.doc.unlockRevision++
				a.status = "Encrypted block unlocked"
			}
			a.closeCrypto()
		}, nil
	})
}
func (a *App) closeCrypto() {
	if a.crypto != nil {
		a.crypto.password = ""
		a.crypto.confirm = ""
		a.crypto.plaintext = ""
	}
	a.crypto = nil
}
func (a *App) encryptedBlock(c *ui.Context, d *Document, payload string) {
	plain, ok := d.Unlocked[payload]
	if !ok {
		if ui.Button(c, a.t("Unlock encrypted block")).Key("unlock:" + payload).Disabled(a.busy).Clicked() {
			a.crypto = &cryptoForm{kind: "decrypt", doc: d, payload: payload}
		}
		return
	}
	ui.Column(c).Border(1, c.Theme().Border).Padding(12).Gap(8).Children(func() {
		ui.RichText(c, searchSpans(plain, d.Query, c.Theme())...).Selectable()
		if ui.Button(c, a.t("Lock")).Clicked() {
			delete(d.Unlocked, payload)
			d.unlockRevision++
		}
	})
}
func (a *App) cryptoModal(c *ui.Context) {
	f := a.crypto
	if f == nil {
		return
	}
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Gap(12).Children(func() {
			title := "Encrypt selection"
			if f.kind == "decrypt" {
				title = "Unlock encrypted block"
			}
			ui.Text(c, a.t(title)).FontSize(20).Bold()
			if f.kind == "encrypt" {
				ui.Text(c, a.t("The password cannot be recovered if you forget it."))
			}
			c.Key("crypto-password")
			ui.TextInput(c, &f.password).Password().Label(a.t("Password")).Disabled(a.busy).AutoFocus()
			if f.kind == "encrypt" {
				c.Key("crypto-confirm")
				ui.TextInput(c, &f.confirm).Password().Label(a.t("Confirm password")).Disabled(a.busy)
			}
			if f.message != "" {
				ui.Text(c, a.t(f.message)).TextColor(c.Theme().Danger)
			}
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, a.t("Cancel")).Disabled(a.busy).Clicked() {
					a.closeCrypto()
				}
				label := "Encrypt"
				if f.kind == "decrypt" {
					label = "Unlock"
				}
				if a.busy {
					label = "Working…"
				}
				if ui.PrimaryButton(c, a.t(label)).Key("crypto-submit").Disabled(a.busy).Clicked() {
					a.submitCrypto()
				}
			})
		})
	})
	if !open && !a.busy {
		a.closeCrypto()
	}
}

func (a *App) lockAll() {
	for _, d := range a.docs {
		if len(d.Unlocked) > 0 {
			d.Unlocked = map[string]string{}
			d.unlockRevision++
		}
	}
}
func (a *App) activate(path string) {
	if a.session.Active != path {
		a.lockAll()
	}
	a.session.Active = path
}
