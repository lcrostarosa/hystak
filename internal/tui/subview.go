package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

// listItem is a generic row in a sub-nav list.
type listItem struct {
	name    string
	columns []string
}

// subView defines how a resource type is displayed and edited in the registry tab.
type subView struct {
	header     string
	loadItems  func(svc *service.Service) []listItem
	addFields  func() []FormField
	editFields func(name string, svc *service.Service) []FormField
	save       func(svc *service.Service, values map[string]string, editName string) error
	delete     func(svc *service.Service, name string) error
}

var subViews = [subNavCount]subView{
	SubNavMCPs:        mcpSubView(),
	SubNavSkills:      skillSubView(),
	SubNavHooks:       hookSubView(),
	SubNavPermissions: permissionSubView(),
	SubNavTemplates:   templateSubView(),
	SubNavPrompts:     promptSubView(),
}

// --- MCPs ---

func mcpSubView() subView {
	return subView{
		header: fmt.Sprintf("  %-20s  %-10s  %s", "NAME", "TRANSPORT", "COMMAND/URL"),
		loadItems: func(svc *service.Service) []listItem {
			servers := svc.ListServers()
			items := make([]listItem, len(servers))
			for i, s := range servers {
				endpoint := s.Command
				if s.Transport == model.TransportSSE || s.Transport == model.TransportHTTP {
					endpoint = s.URL
				}
				items[i] = listItem{
					name:    s.Name,
					columns: []string{truncate(s.Name, 20), string(s.Transport), truncate(endpoint, 40)},
				}
			}
			return items
		},
		addFields: func() []FormField {
			return []FormField{
				{Label: "Name", Placeholder: "server-name"},
				{Label: "Transport", Placeholder: "stdio | sse | http", Value: "stdio"},
				{Label: "Command", Placeholder: "npx"},
				{Label: "Args", Placeholder: "-y, @anthropic/mcp-github"},
				{Label: "URL", Placeholder: "https://..."},
				{Label: "Env", Placeholder: "KEY=val, KEY2=val2"},
				{Label: "Description", Placeholder: "optional description"},
			}
		},
		editFields: func(name string, svc *service.Service) []FormField {
			srv, ok := svc.GetServer(name)
			if !ok {
				return nil
			}
			return []FormField{
				{Label: "Name", Value: srv.Name},
				{Label: "Transport", Value: string(srv.Transport)},
				{Label: "Command", Value: srv.Command},
				{Label: "Args", Value: strings.Join(srv.Args, ", ")},
				{Label: "URL", Value: srv.URL},
				{Label: "Env", Value: formatEnv(srv.Env)},
				{Label: "Description", Value: srv.Description},
			}
		},
		save: func(svc *service.Service, values map[string]string, editName string) error {
			transport := model.Transport(strings.TrimSpace(values["Transport"]))
			if !transport.Valid() {
				return fmt.Errorf("invalid transport %q", transport)
			}
			srv := model.ServerDef{
				Name:        strings.TrimSpace(values["Name"]),
				Transport:   transport,
				Command:     strings.TrimSpace(values["Command"]),
				Args:        parseCSV(values["Args"]),
				URL:         strings.TrimSpace(values["URL"]),
				Env:         parseKV(values["Env"]),
				Description: strings.TrimSpace(values["Description"]),
			}
			if editName != "" {
				return svc.UpdateServer(srv)
			}
			return svc.AddServer(srv)
		},
		delete: func(svc *service.Service, name string) error {
			return svc.DeleteServer(name)
		},
	}
}

// --- Skills ---

func skillSubView() subView {
	return subView{
		header: fmt.Sprintf("  %-20s  %-30s  %s", "NAME", "DESCRIPTION", "SOURCE"),
		loadItems: func(svc *service.Service) []listItem {
			skills := svc.ListSkills()
			items := make([]listItem, len(skills))
			for i, s := range skills {
				items[i] = listItem{
					name:    s.Name,
					columns: []string{truncate(s.Name, 20), truncate(s.Description, 30), truncate(s.Source, 30)},
				}
			}
			return items
		},
		addFields: func() []FormField {
			return []FormField{
				{Label: "Name", Placeholder: "skill-name"},
				{Label: "Description", Placeholder: "what this skill does"},
				{Label: "Source", Placeholder: "/path/to/SKILL.md"},
			}
		},
		editFields: func(name string, svc *service.Service) []FormField {
			s, ok := svc.GetSkill(name)
			if !ok {
				return nil
			}
			return []FormField{
				{Label: "Name", Value: s.Name},
				{Label: "Description", Value: s.Description},
				{Label: "Source", Value: s.Source},
			}
		},
		save: func(svc *service.Service, values map[string]string, editName string) error {
			skill := model.SkillDef{
				Name:        strings.TrimSpace(values["Name"]),
				Description: strings.TrimSpace(values["Description"]),
				Source:      strings.TrimSpace(values["Source"]),
			}
			if editName != "" {
				return svc.UpdateSkill(skill)
			}
			return svc.AddSkill(skill)
		},
		delete: func(svc *service.Service, name string) error {
			return svc.DeleteSkill(name)
		},
	}
}

// --- Hooks ---

func hookSubView() subView {
	return subView{
		header: fmt.Sprintf("  %-20s  %-14s  %-16s  %s", "NAME", "EVENT", "MATCHER", "COMMAND"),
		loadItems: func(svc *service.Service) []listItem {
			hooks := svc.ListHooks()
			items := make([]listItem, len(hooks))
			for i, h := range hooks {
				items[i] = listItem{
					name:    h.Name,
					columns: []string{truncate(h.Name, 20), string(h.Event), truncate(h.Matcher, 16), truncate(h.Command, 30)},
				}
			}
			return items
		},
		addFields: func() []FormField {
			return []FormField{
				{Label: "Name", Placeholder: "hook-name"},
				{Label: "Event", Placeholder: "PreToolUse | PostToolUse | Notification | Stop"},
				{Label: "Matcher", Placeholder: "Bash"},
				{Label: "Command", Placeholder: "eslint --fix"},
				{Label: "Timeout", Placeholder: "30", Value: "30"},
			}
		},
		editFields: func(name string, svc *service.Service) []FormField {
			h, ok := svc.GetHook(name)
			if !ok {
				return nil
			}
			return []FormField{
				{Label: "Name", Value: h.Name},
				{Label: "Event", Value: string(h.Event)},
				{Label: "Matcher", Value: h.Matcher},
				{Label: "Command", Value: h.Command},
				{Label: "Timeout", Value: strconv.Itoa(h.Timeout)},
			}
		},
		save: func(svc *service.Service, values map[string]string, editName string) error {
			event := model.HookEvent(strings.TrimSpace(values["Event"]))
			if !event.Valid() {
				return fmt.Errorf("invalid hook event %q", event)
			}
			timeout, err := strconv.Atoi(strings.TrimSpace(values["Timeout"]))
			if err != nil {
				return fmt.Errorf("invalid timeout %q: %w", values["Timeout"], err)
			}
			if timeout <= 0 {
				return fmt.Errorf("timeout must be positive, got %d", timeout)
			}
			hook := model.HookDef{
				Name:    strings.TrimSpace(values["Name"]),
				Event:   event,
				Matcher: strings.TrimSpace(values["Matcher"]),
				Command: strings.TrimSpace(values["Command"]),
				Timeout: timeout,
			}
			if editName != "" {
				return svc.UpdateHook(hook)
			}
			return svc.AddHook(hook)
		},
		delete: func(svc *service.Service, name string) error {
			return svc.DeleteHook(name)
		},
	}
}

// --- Permissions ---

func permissionSubView() subView {
	return subView{
		header: fmt.Sprintf("  %-20s  %-30s  %s", "NAME", "RULE", "TYPE"),
		loadItems: func(svc *service.Service) []listItem {
			perms := svc.ListPermissions()
			items := make([]listItem, len(perms))
			for i, p := range perms {
				items[i] = listItem{
					name:    p.Name,
					columns: []string{truncate(p.Name, 20), truncate(p.Rule, 30), string(p.Type)},
				}
			}
			return items
		},
		addFields: func() []FormField {
			return []FormField{
				{Label: "Name", Placeholder: "rule-name"},
				{Label: "Rule", Placeholder: "Bash(*)"},
				{Label: "Type", Placeholder: "allow | deny", Value: "allow"},
			}
		},
		editFields: func(name string, svc *service.Service) []FormField {
			p, ok := svc.GetPermission(name)
			if !ok {
				return nil
			}
			return []FormField{
				{Label: "Name", Value: p.Name},
				{Label: "Rule", Value: p.Rule},
				{Label: "Type", Value: string(p.Type)},
			}
		},
		save: func(svc *service.Service, values map[string]string, editName string) error {
			permType := model.PermissionType(strings.TrimSpace(values["Type"]))
			if !permType.Valid() {
				return fmt.Errorf("invalid permission type %q", permType)
			}
			perm := model.PermissionRule{
				Name: strings.TrimSpace(values["Name"]),
				Rule: strings.TrimSpace(values["Rule"]),
				Type: permType,
			}
			if editName != "" {
				return svc.UpdatePermission(perm)
			}
			return svc.AddPermission(perm)
		},
		delete: func(svc *service.Service, name string) error {
			return svc.DeletePermission(name)
		},
	}
}

// --- Templates ---

func templateSubView() subView {
	return subView{
		header: fmt.Sprintf("  %-20s  %s", "NAME", "SOURCE"),
		loadItems: func(svc *service.Service) []listItem {
			tmpls := svc.ListTemplates()
			items := make([]listItem, len(tmpls))
			for i, t := range tmpls {
				items[i] = listItem{
					name:    t.Name,
					columns: []string{truncate(t.Name, 20), truncate(t.Source, 50)},
				}
			}
			return items
		},
		addFields: func() []FormField {
			return []FormField{
				{Label: "Name", Placeholder: "template-name"},
				{Label: "Source", Placeholder: "/path/to/template.md"},
			}
		},
		editFields: func(name string, svc *service.Service) []FormField {
			tmpl, ok := svc.GetTemplate(name)
			if !ok {
				return nil
			}
			return []FormField{
				{Label: "Name", Value: tmpl.Name},
				{Label: "Source", Value: tmpl.Source},
			}
		},
		save: func(svc *service.Service, values map[string]string, editName string) error {
			tmpl := model.TemplateDef{
				Name:   strings.TrimSpace(values["Name"]),
				Source: strings.TrimSpace(values["Source"]),
			}
			if editName != "" {
				return svc.UpdateTemplate(tmpl)
			}
			return svc.AddTemplate(tmpl)
		},
		delete: func(svc *service.Service, name string) error {
			return svc.DeleteTemplate(name)
		},
	}
}

// --- Prompts ---

func promptSubView() subView {
	return subView{
		header: fmt.Sprintf("  %-20s  %-16s  %-6s  %s", "NAME", "CATEGORY", "ORDER", "TAGS"),
		loadItems: func(svc *service.Service) []listItem {
			prompts := svc.ListPrompts()
			items := make([]listItem, len(prompts))
			for i, p := range prompts {
				tags := strings.Join(p.Tags, ", ")
				items[i] = listItem{
					name:    p.Name,
					columns: []string{truncate(p.Name, 20), truncate(p.Category, 16), strconv.Itoa(p.Order), truncate(tags, 20)},
				}
			}
			return items
		},
		addFields: func() []FormField {
			return []FormField{
				{Label: "Name", Placeholder: "prompt-name"},
				{Label: "Description", Placeholder: "what this prompt does"},
				{Label: "Source", Placeholder: "/path/to/prompt.md"},
				{Label: "Category", Placeholder: "safety"},
				{Label: "Order", Placeholder: "10", Value: "10"},
				{Label: "Tags", Placeholder: "tag1, tag2"},
			}
		},
		editFields: func(name string, svc *service.Service) []FormField {
			p, ok := svc.GetPrompt(name)
			if !ok {
				return nil
			}
			return []FormField{
				{Label: "Name", Value: p.Name},
				{Label: "Description", Value: p.Description},
				{Label: "Source", Value: p.Source},
				{Label: "Category", Value: p.Category},
				{Label: "Order", Value: strconv.Itoa(p.Order)},
				{Label: "Tags", Value: strings.Join(p.Tags, ", ")},
			}
		},
		save: func(svc *service.Service, values map[string]string, editName string) error {
			order, err := strconv.Atoi(strings.TrimSpace(values["Order"]))
			if err != nil {
				return fmt.Errorf("invalid order %q: %w", values["Order"], err)
			}
			prompt := model.PromptDef{
				Name:        strings.TrimSpace(values["Name"]),
				Description: strings.TrimSpace(values["Description"]),
				Source:      strings.TrimSpace(values["Source"]),
				Category:    strings.TrimSpace(values["Category"]),
				Order:       order,
				Tags:        parseCSV(values["Tags"]),
			}
			if editName != "" {
				return svc.UpdatePrompt(prompt)
			}
			return svc.AddPrompt(prompt)
		},
		delete: func(svc *service.Service, name string) error {
			return svc.DeletePrompt(name)
		},
	}
}
