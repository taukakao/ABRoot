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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vanilla-os/abroot/core"
	"github.com/vanilla-os/orchid/cmdr"
)

type DiskLayoutError struct {
	Device string
}

func (e *DiskLayoutError) Error() string {
	return fmt.Sprintf("device %s has an unsupported layout", e.Device)
}

func NewMountSysCommand() *cmdr.Command {
	cmd := cmdr.NewCommand(
		"mount-sys",
		"",
		"",
		mountSysCmd,
	)

	cmd.WithBoolFlag(
		cmdr.NewBoolFlag(
			"dry-run",
			"d",
			abroot.Trans("mount-sys.dryRunFlag"),
			false))

	cmd.Example = "abroot mount-sys"

	cmd.Hidden = true

	return cmd
}

// helper function which only returns syntax errors and prints other ones
func mountSysCmd(cmd *cobra.Command, args []string) error {
	err := mountSys(cmd, args)
	if err != nil {
		cmdr.Error.Println(err)
		os.Exit(1)
	}
	return nil
}

func mountSys(cmd *cobra.Command, args []string) error {
	// if !core.RootCheck(false) {
	// 	cmdr.Error.Println(abroot.Trans("status.rootRequired"))
	// 	return nil
	// }

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	manager := core.NewABRootManager()
	present, err := manager.GetPresent()
	if err != nil {
		return err
	}

	err = mountVar(manager.VarPartition, dryRun)
	if err != nil {
		return err
	}

	err = mountBindMounts(dryRun)
	if err != nil {
		return err
	}

	err = mountOverlayMounts(present.Label, dryRun)
	if err != nil {
		return err
	}

	err = adjustFstab(dryRun)
	if err != nil {
		return err
	}

	return nil
}

func mountVar(varPart core.Partition, dryRun bool) error {
	if varPart.IsEncrypted() {
		unlockVar(varPart, "TODO")
	}

	if varPart.IsDevMapper() {
		dev := varPart.Device
		cmdr.Info.Println("mount /dev/mapper/" + dev + " /var")
	} else {
		dev := varPart.Device
		cmdr.Info.Println("mount /dev/" + dev + " /var")
	}

	return nil
}

func mountBindMounts(dryRun bool) error {
	type bindMount struct {
		from, to string
	}

	binds := []bindMount{{"/var/home", "/home"}, {"/var/opt", "/opt"}}

	for _, bind := range binds {
		cmdr.Info.Println("mount --bind " + bind.from + " " + bind.to)
	}

	return nil
}

func mountOverlayMounts(rootLabel string, dryRun bool) error {
	type overlayMount struct {
		destination       string
		lowerdirs         []string
		upperdir, workdir string
	}

	overlays := []overlayMount{{"/.system/etc", []string{"/.system/etc"}, "/var/lib/abroot/etc/" + rootLabel, "/var/lib/abroot/etc/" + rootLabel + "-work"}}

	for _, overlay := range overlays {
		lowerCombined := strings.Join(overlay.lowerdirs, ":")
		options := "lowerdir=" + lowerCombined + ",upperdir=" + overlay.upperdir + ",workdir=" + overlay.workdir
		cmdr.Info.Println("mount -t overlay overlay -o " + options + " " + overlay.destination)

		// err := syscall.Mount("overlay", overlay.destination, "overlay", 0, options)
	}
	return nil
}

func adjustFstab(dryRun bool) error {
	cmdr.Info.Println("switch the root in fstab")
	return nil
}

func unlockVar(varPart core.Partition, passphrase string) error {
	if !varPart.IsEncrypted() {
		return &DiskLayoutError{varPart.Device}
	}
	if !varPart.IsDevMapper() {
		return &DiskLayoutError{varPart.Device}
	}

	dev := varPart.Device
	uuid := varPart.Uuid
	cmdr.Info.Println("cryptsetup luksOpen /dev/mapper/" + dev + " luks-" + uuid)

	return nil
}

// func getCurrentlyBootedPartition(a *core.ABRootManager) (string, string, error) {
// 	bootPart, err := a.GetBoot()
// 	if err != nil {
// 		return "", "", err
// 	}
// 	uuid := uuid.New().String()
// 	tmpBootMount := filepath.Join("/tmp", uuid)
// 	err = os.Mkdir(tmpBootMount, 0o755)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	err = bootPart.Mount(tmpBootMount)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	defer bootPart.Unmount()

// 	g, err := core.NewGrub(bootPart)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	isPresent, err := g.IsBootedIntoPresentRoot()
// 	if err != nil {
// 		return "", "", err
// 	}

// 	presentMark := ""
// 	futureMark := ""
// 	if isPresent {
// 		presentMark = " ✓"
// 	} else {
// 		futureMark = " ✓"
// 	}

// 	return presentMark, futureMark, nil
// }
