package main

import (
	"fmt"
	"strconv"

	vscale "github.com/dpvpro/vscale-api-client-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"

	"time"
)

func resourceScalet() *schema.Resource {
	return &schema.Resource{
		Create: resourceScaletCreate,
		Read:   resourceScaletRead,
		Exists: resourceScaletExists,
		// Update: resourceScaletUpdate,
		Delete: resourceScaletDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"make_from": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rplan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ssh_keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"public_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceScaletCreate(d *schema.ResourceData, m any) error {
	client := m.(*vscale.WebClient)

	name := d.Get("name").(string)
	from := d.Get("make_from").(string)
	plan := d.Get("rplan").(string)
	location := d.Get("location").(string)

	var keyIDS []int64

	sshKeysCount := d.Get("ssh_keys.#").(int)
	if sshKeysCount > 0 {
		remoteSSHKeys, _, err := client.SSHKey.List()
		if err != nil {
			return errors.Wrap(err, "getting ssh keys failed")
		}

		if remoteSSHKeys == nil || len(*remoteSSHKeys) == 0 {
			return errors.New("there are no remote ssh keys")
		}

		for i := 0; i < sshKeysCount; i++ {
			key := fmt.Sprintf("ssh_keys.%d", i)
			keyRef := d.Get(key).(string)
			for _, remoteKey := range *remoteSSHKeys {
				keyID, err := strconv.ParseInt(keyRef, 10, 64)
				if err != nil {
					continue
				}

				if remoteKey.ID == keyID {
					keyIDS = append(keyIDS, remoteKey.ID)
				}
			}
		}
	}

	scalet, _, err := client.Scalet.CreateWithoutPassword(from, plan, name, location, true, keyIDS, true)
	if err != nil {
		return errors.Wrap(err, "creating scalet failed")
	}

	time.Sleep(15 * time.Second)

	publicAddress, err := findPublicAddress(client, scalet.CTID)
	if err != nil {
		return errors.Wrap(err, "search of public address failed")
	}

	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": publicAddress,
	})

	d.Set("public_address", publicAddress)
	d.SetId(strconv.FormatInt(scalet.CTID, 10))

	return nil
}

func resourceScaletRead(d *schema.ResourceData, m any) error {
	client := m.(*vscale.WebClient)

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid ctid")
	}

	scalet, _, err := client.Scalet.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting scalet failed")
	}

	d.Set("name", scalet.Name)
	d.Set("make_from", scalet.MadeFrom)
	d.Set("rplan", scalet.Rplan)
	d.Set("location", scalet.Location)

	return nil
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}

	return false
}

func resourceScaletExists(d *schema.ResourceData, m any) (bool, error) {
	client := m.(*vscale.WebClient)

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return true, errors.Wrap(err, "invalid ctid")
	}

	_, _, err = client.Scalet.Get(id)
	if err != nil {
		return true, errors.Wrap(err, "getting scalet failed")
	}

	return true, nil
}

func resourceScaletUpdate(d *schema.ResourceData, m any) error {
	return nil
}

func resourceScaletDelete(d *schema.ResourceData, m any) error {
	client := m.(*vscale.WebClient)

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid ctid")
	}

	_, _, err = client.Scalet.Remove(id, true)
	if err != nil {
		return errors.Wrap(err, "removing scalet failed")
	}

	return nil
}

func findPublicAddress(client *vscale.WebClient, scaletID int64) (string, error) {
	scalet, _, err := client.Scalet.Get(scaletID)
	if err != nil {
		return "", errors.Wrap(err, "getting scalet failed")
	}

	return scalet.PublicAddresses.Address, nil
}
