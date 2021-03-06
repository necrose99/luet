// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/mudler/luet/pkg/config"
	installer "github.com/mudler/luet/pkg/installer"
	. "github.com/mudler/luet/pkg/logger"
	pkg "github.com/mudler/luet/pkg/package"

	_gentoo "github.com/Sabayon/pkgs-checker/pkg/gentoo"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <pkg> <pkg2> ...",
	Short: "Uninstall a package or a list of packages",
	Long:  `Uninstall packages`,
	PreRun: func(cmd *cobra.Command, args []string) {
		LuetCfg.Viper.BindPFlag("system.database_path", cmd.Flags().Lookup("system-dbpath"))
		LuetCfg.Viper.BindPFlag("system.rootfs", cmd.Flags().Lookup("system-target"))
		LuetCfg.Viper.BindPFlag("solver.type", cmd.Flags().Lookup("solver-type"))
		LuetCfg.Viper.BindPFlag("solver.discount", cmd.Flags().Lookup("solver-discount"))
		LuetCfg.Viper.BindPFlag("solver.rate", cmd.Flags().Lookup("solver-rate"))
		LuetCfg.Viper.BindPFlag("solver.max_attempts", cmd.Flags().Lookup("solver-attempts"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		var systemDB pkg.PackageDatabase

		for _, a := range args {
			gp, err := _gentoo.ParsePackageStr(a)
			if err != nil {
				Fatal("Invalid package string ", a, ": ", err.Error())
			}
			if gp.Version == "" {
				gp.Version = "0"
				gp.Condition = _gentoo.PkgCondGreaterEqual
			}
			pack := &pkg.DefaultPackage{
				Name: gp.Name,
				Version: fmt.Sprintf("%s%s%s",
					pkg.PkgSelectorConditionFromInt(gp.Condition.Int()).String(),
					gp.Version,
					gp.VersionSuffix,
				),
				Category: gp.Category,
				Uri:      make([]string, 0),
			}

			stype := LuetCfg.Viper.GetString("solver.type")
			discount := LuetCfg.Viper.GetFloat64("solver.discount")
			rate := LuetCfg.Viper.GetFloat64("solver.rate")
			attempts := LuetCfg.Viper.GetInt("solver.max_attempts")

			LuetCfg.GetSolverOptions().Type = stype
			LuetCfg.GetSolverOptions().LearnRate = float32(rate)
			LuetCfg.GetSolverOptions().Discount = float32(discount)
			LuetCfg.GetSolverOptions().MaxAttempts = attempts

			Debug("Solver", LuetCfg.GetSolverOptions().CompactString())

			inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{Concurrency: LuetCfg.GetGeneral().Concurrency, SolverOptions: *LuetCfg.GetSolverOptions()})

			if LuetCfg.GetSystem().DatabaseEngine == "boltdb" {
				systemDB = pkg.NewBoltDatabase(
					filepath.Join(LuetCfg.GetSystem().GetSystemRepoDatabaseDirPath(), "luet.db"))
			} else {
				systemDB = pkg.NewInMemoryDatabase(true)
			}
			system := &installer.System{Database: systemDB, Target: LuetCfg.GetSystem().Rootfs}
			err = inst.Uninstall(pack, system)
			if err != nil {
				Fatal("Error: " + err.Error())
			}
		}
	},
}

func init() {
	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	uninstallCmd.Flags().String("system-dbpath", path, "System db path")
	uninstallCmd.Flags().String("system-target", path, "System rootpath")
	uninstallCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+AvailableResolvers+" )")
	uninstallCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	uninstallCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	uninstallCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	RootCmd.AddCommand(uninstallCmd)
}
