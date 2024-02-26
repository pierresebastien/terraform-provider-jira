package jira

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

const actorTypeUser = "atlassian-user-role-actor"
const actorTypeGroup = "atlassian-group-role-actor"

// ProjectMembershipRequest represents a JIRA ProjectMembership
type ProjectMembershipRequest struct {
	ID    int      `json:"id,omitempty"`
	User  []string `json:"user,omitempty"`
	Group []string `json:"group,omitempty"`
}

// ProjectMembership represents a JIRA ProjectMembership
type ActorGroup struct {
	GroupId string `json:"groupId"`
	Name    string `json:"name"`
}

// ProjectMembership represents a JIRA ProjectMembership
type ActorUser struct {
	AccountId string `json:"accountId"`
}

// ProjectMembership represents a JIRA ProjectMembership
type ProjectMembership struct {
	ID    int        `json:"id,omitempty"`
	Type  string     `json:"type"`
	Group ActorGroup `json:"actorGroup,omitempty"`
	User  ActorUser  `json:"actorUser,omitempty"`
}

// ProjectRole represents the actors of a Role within a Project
type ProjectRole struct {
	Actors []ProjectMembership `json:"actors"`
}

func resourceProjectMembership() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectMembershipCreate,
		Read:   resourceProjectMembershipRead,
		Delete: resourceProjectMembershipDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"project_key": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"role_id": &schema.Schema{
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"account_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"group"},
			},
			"group": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"account_id"},
			},
		},
	}
}

func setProjectMembershipResource(w *ProjectMembership, d *schema.ResourceData) {

	if w.Type == actorTypeUser {
		d.Set("account_id", w.User.AccountId)
	} else if w.Type == actorTypeGroup {
		d.Set("group", w.Group.Name)
	}
}

func setProjectMembership(w *ProjectMembershipRequest, d *schema.ResourceData) error {

	if name, ok := d.GetOk("account_id"); ok {
		w.User = []string{name.(string)}
	} else if name, ok := d.GetOk("group"); ok {
		w.Group = []string{name.(string)}
	} else {
		return errors.New("Neither account_id nor group is set")
	}

	return nil
}

// resourceProjectMembershipCreate creates a new jira issue using the jira api
func resourceProjectMembershipCreate(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)
	projectKey := d.Get("project_key").(string)
	roleID := d.Get("role_id").(int)

	role := new(ProjectMembershipRequest)
	returnedRole := new(ProjectRole)

	err := setProjectMembership(role, d)
	if err != nil {
		return err
	}

	urlStr := fmt.Sprintf("%s/%d", projectRoleAPIEndpoint(projectKey), roleID)

	err = request(config.jiraClient, "POST", urlStr, role, returnedRole)
	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	d.SetId(strconv.Itoa(returnedRole.Actors[0].ID))

	return resourceProjectMembershipRead(d, m)
}

// resourceProjectMembershipRead
func resourceProjectMembershipRead(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	projectKey := d.Get("project_key").(string)
	roleID := d.Get("role_id").(int)

	urlStr := fmt.Sprintf("%s/%d", projectRoleAPIEndpoint(projectKey), roleID)

	role := new(ProjectRole)
	err := request(config.jiraClient, "GET", urlStr, nil, role)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	for _, actor := range role.Actors {
		if strconv.Itoa(actor.ID) == d.Id() {

			return nil
		}
	}

	d.SetId("")
	return nil
}

// resourceProjectMembershipUpdate updates jira issue using jira api

// resourceProjectMembershipDelete deletes jira issue using the jira api
func resourceProjectMembershipDelete(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	projectKey := d.Get("project_key").(string)
	roleID := d.Get("role_id").(int)

	var urlStr string

	if account_id, ok := d.GetOk("account_id"); ok {
		urlStr = fmt.Sprintf("%s/%d?user=%s", projectRoleAPIEndpoint(projectKey), roleID, url.QueryEscape(account_id.(string)))

	} else if group, ok := d.GetOk("group"); ok {
		urlStr = fmt.Sprintf("%s/%d?group=%s", projectRoleAPIEndpoint(projectKey), roleID, url.QueryEscape(group.(string)))
	} else {
		return nil
	}

	err := request(config.jiraClient, "DELETE", urlStr, nil, nil)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	return nil
}
