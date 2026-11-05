// gitclt
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/access_mgmt/users/sshkeys.go
// Original timestamp : 2026.11.04 21:00:00

package users

import (
	"context"
	"fmt"
	"gitclt/forge"
	"gitclt/shared"
	"net/http"
	"os"
	"strings"

	cerr "github.com/jeanfrancoisgratton/customError/v3"
	gol "github.com/jeanfrancoisgratton/gitoperationslib"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// ListSSHKeys lists the public SSH keys of the connection's user
func ListSSHKeys(displayOutput bool) ([]gol.SSHKey, *cerr.CustomError) {
	ctx := context.Background()

	_, acct, ce := connectAccount(ctx)
	if ce != nil {
		return nil, ce
	}
	list, ce := fetchSSHKeys(ctx, acct)
	if ce != nil {
		return nil, ce
	}

	// we only want the data, we do not need it sent to stdout
	if !displayOutput {
		return list, nil
	}

	if JSONoutput {
		return list, shared.PrintJSON(list, "SSH key list")
	}
	if shared.QuietOutput {
		for _, k := range list {
			fmt.Println(k.Title)
		}
		return list, nil
	}

	renderSSHKeyTable(list)
	return list, nil
}

// GetSSHKey shows a public SSH key of the connection's user, found by title, or by ID if --id was passed
func GetSSHKey(title string, displayOutput bool) (*gol.SSHKey, *cerr.CustomError) {
	ctx := context.Background()

	_, acct, ce := connectAccount(ctx)
	if ce != nil {
		return nil, ce
	}
	list, ce := fetchSSHKeys(ctx, acct)
	if ce != nil {
		return nil, ce
	}
	key, ce := findSSHKey(list, title)
	if ce != nil {
		return nil, ce
	}

	if !displayOutput {
		return key, nil
	}
	if JSONoutput {
		return key, shared.PrintJSON(key, "SSH key")
	}
	printSSHKey(key)
	return key, nil
}

// AddSSHKey adds the public SSH key found in keyFile to the connection's user
func AddSSHKey(title, keyFile string) *cerr.CustomError {
	ctx := context.Background()

	// Read first: nothing leaves this machine unless it is known to be a public key
	publicKey, ce := readPublicKey(keyFile)
	if ce != nil {
		return ce
	}

	_, acct, ce := connectAccount(ctx)
	if ce != nil {
		return ce
	}

	key, e := acct.SSHKeys().Add(ctx, gol.AddSSHKeyOptions{Title: title, Key: publicKey})
	if e != nil {
		return forge.Error("Unable to add the SSH key", e)
	}

	if JSONoutput {
		return shared.PrintJSON(key, "SSH key")
	}
	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Added SSH key " + hftx.Brown(key.Title)))
	}
	return nil
}

// DeleteSSHKey removes a public SSH key of the connection's user, found by title, or by ID if --id was passed
func DeleteSSHKey(title string) *cerr.CustomError {
	ctx := context.Background()

	_, acct, ce := connectAccount(ctx)
	if ce != nil {
		return ce
	}
	list, ce := fetchSSHKeys(ctx, acct)
	if ce != nil {
		return ce
	}
	key, ce := findSSHKey(list, title)
	if ce != nil {
		return ce
	}

	if !confirmed("Delete SSH key " + hftx.Brown(key.Title) + " ? [y/N]: ") {
		fmt.Println("Aborted, SSH key " + hftx.Brown(key.Title) + " was not deleted")
		return nil
	}

	if e := acct.SSHKeys().Delete(ctx, key.ID); e != nil {
		return forge.Error("Unable to delete the SSH key", e)
	}

	if !shared.QuietOutput {
		fmt.Println(hftx.EnabledSign("Deleted SSH key " + hftx.Brown(key.Title)))
	}
	return nil
}

func fetchSSHKeys(ctx context.Context, acct gol.Account) ([]gol.SSHKey, *cerr.CustomError) {
	list, e := shared.FetchAll(ctx, acct.SSHKeys().List)
	if e != nil {
		return nil, forge.Error("Unable to fetch the SSH key list", e)
	}
	return list, nil
}

// Finds the key with this title, or the one with SSHKeyID if --id was passed
// GitHub lets several keys share a title: the title is then refused, as a delete could hit the wrong key
func findSSHKey(list []gol.SSHKey, title string) (*gol.SSHKey, *cerr.CustomError) {
	var match *gol.SSHKey
	var ids []string
	for i := range list {
		if (SSHKeyID != 0 && list[i].ID == SSHKeyID) || (SSHKeyID == 0 && list[i].Title == title) {
			match = &list[i]
			ids = append(ids, fmt.Sprintf("%d", list[i].ID))
		}
	}
	if match == nil {
		what := "title " + title
		if SSHKeyID != 0 {
			what = fmt.Sprintf("ID %d", SSHKeyID)
		}
		return nil, &cerr.CustomError{Title: "Unable to find the SSH key", Message: "There is no SSH key with " + what, Code: http.StatusNotFound}
	}
	if len(ids) > 1 {
		return nil, &cerr.CustomError{Title: "Ambiguous SSH key",
			Message: fmt.Sprintf("%d SSH keys have the title %s; designate one with --id (%s)", len(ids), title, strings.Join(ids, ", "))}
	}
	return match, nil
}

// Reads a public key from a file, and refuses anything else: passing the private key by mistake must not send it
func readPublicKey(path string) (string, *cerr.CustomError) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", &cerr.CustomError{Title: "Unable to read the SSH key", Message: e.Error()}
	}
	key := strings.TrimSpace(string(b))
	if !isPublicKey(key) {
		return "", &cerr.CustomError{Title: "Invalid SSH key",
			Message: path + " does not hold one public SSH key (ex: the content of id_ed25519.pub); nothing was sent"}
	}
	return key, nil
}

// A single "<type> <base64> [comment]" line; the server does the actual validation
func isPublicKey(key string) bool {
	fields := strings.Fields(key)
	if len(fields) < 2 || strings.ContainsAny(key, "\r\n") {
		return false
	}
	for _, prefix := range []string{"ssh-", "ecdsa-", "sk-"} {
		if strings.HasPrefix(fields[0], prefix) {
			return true
		}
	}
	return false
}
