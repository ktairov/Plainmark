package preview

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func ResolveLink(current, href string) (local, anchor, external string, err error) {
	u, err := url.Parse(href)
	if err != nil {
		return "", "", "", err
	}
	if u.Scheme != "" {
		if u.Scheme == "https" || u.Scheme == "http" || u.Scheme == "mailto" {
			return "", "", href, nil
		}
		return "", "", "", fmt.Errorf("unsupported link scheme")
	}
	if u.Host != "" {
		return "", "", "", fmt.Errorf("unsupported link")
	}
	anchor = u.Fragment
	if u.Path == "" {
		return "", anchor, "", nil
	}
	linked := strings.ReplaceAll(u.Path, "\\", "/")
	ext := strings.ToLower(path.Ext(linked))
	if ext != ".md" && ext != ".mdx" {
		return "", "", "", fmt.Errorf("link is not a Markdown note")
	}
	var parts []string
	if !strings.HasPrefix(linked, "/") {
		dir := path.Dir(current)
		if dir != "." {
			parts = strings.Split(dir, "/")
		}
	}
	for _, part := range strings.Split(linked, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(parts) == 0 {
				return "", "", "", fmt.Errorf("link leaves workspace")
			}
			parts = parts[:len(parts)-1]
		default:
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "/"), anchor, "", nil
}
