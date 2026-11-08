package blupcloud

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
)

// Vault names are sealed bound to the vault's id (cloudcrypt.SealName,
// format 2), so the cloud can't show one vault's name on another. A vault
// gets its id only when it is created, so it is created with a format 1
// name and named again with the format 2 one right away. Names sealed
// before format 2 still open (cloudcrypt.OpenName says they are legacy) and
// are sealed again the next time the device sends a name anyway: a rename,
// or linking the vault on a device (UpgradeVaultName). Opening one never
// writes anything.

// CreateNamedVault makes a vault named title, bound to its id. When the
// second step fails the vault still exists with the format 1 name, which
// opens and is sealed again at the next rename or link.
func (c *Client) CreateNamedVault(ctx context.Context, keys *cloudcrypt.Keys, title string) (Vault, error) {
	legacy, err := keys.SealLegacyName(title)
	if err != nil {
		return Vault{}, err
	}
	v, err := c.CreateVault(ctx, legacy)
	if err != nil {
		return v, err
	}
	if name, err := keys.SealName(v.ID, title); err == nil && c.RenameVault(ctx, v.ID, name) == nil {
		v.Name = name
	}
	return v, nil
}

// RenameSealed gives the vault id the name title, sealed bound to its id.
func (c *Client) RenameSealed(ctx context.Context, keys *cloudcrypt.Keys, id, title string) error {
	if !ValidVaultID(id) {
		return ErrVaultMissing
	}
	name, err := keys.SealName(id, title)
	if err != nil {
		return err
	}
	return c.RenameVault(ctx, id, name)
}

// UpgradeVaultName seals the name of vault id again, bound to its id, when
// the cloud still has it in format 1. It is called once, when a device
// links the vault; a name already in format 2 (or one that doesn't open)
// is left as it is.
func (c *Client) UpgradeVaultName(ctx context.Context, keys *cloudcrypt.Keys, id string) error {
	vaults, err := c.Vaults(ctx)
	if err != nil {
		return err
	}
	for _, v := range vaults {
		if v.ID != id {
			continue
		}
		title, legacy, err := keys.OpenName(v.ID, v.Name)
		if err != nil || !legacy {
			return nil
		}
		return c.RenameSealed(ctx, keys, v.ID, title)
	}
	return nil
}
