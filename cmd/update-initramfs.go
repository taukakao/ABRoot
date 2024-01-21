package cmd

/*	License: GPLv3
	Authors:
		Mirko Brombin <mirko@fabricators.ltd>
		Vanilla OS Contributors <https://github.com/vanilla-os/>
	Copyright: 2024
	Description:
		ABRoot is utility which provides full immutability and
		atomicity to a Linux system, by transacting between
		two root filesystems. Updates are performed using OCI
		images, to ensure that the system is always in a
		consistent state.
*/

import (
	"github.com/spf13/cobra"

	"github.com/vanilla-os/abroot/core"
	"github.com/vanilla-os/orchid/cmdr"
)

func NewUpdateInitfsCommand() *cmdr.Command {
	cmd := cmdr.NewCommand(
		"update-initramfs",
		abroot.Trans("updateInitramfs.long"),
		abroot.Trans("updateInitramfs.short"),
		updateInitramfs,
	)

	cmd.WithBoolFlag(
		cmdr.NewBoolFlag(
			"dry-run",
			"d",
			abroot.Trans("updateInitramfs.dryRunFlag"),
			false))

	cmd.Example = "abroot update-initramfs"

	return cmd
}

func updateInitramfs(cmd *cobra.Command, args []string) error {
	if !core.RootCheck(false) {
		cmdr.Error.Println(abroot.Trans("updateInitramfs.rootRequired"))
		return nil
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		cmdr.Error.Println(err)
		return err
	}

	aBsys, err := core.NewABSystem()
	if err != nil {
		cmdr.Error.Println(err)
		return err
	}

	if dryRun {
		err = aBsys.RunOperation(core.DRY_RUN_INITRAMFS)
	} else {
		err = aBsys.RunOperation(core.INITRAMFS)
	}
	if err != nil {
		cmdr.Error.Printf(abroot.Trans("updateInitramfs.updateFailed"), err)
		return err
	}

	cmdr.Info.Println(abroot.Trans("updateInitramfs.updateSuccess"))

	return nil
}
