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
	"os"
	"path/filepath"

	. "github.com/mudler/luet/pkg/config"
	installer "github.com/mudler/luet/pkg/installer"
	. "github.com/mudler/luet/pkg/logger"
	pkg "github.com/mudler/luet/pkg/package"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades the system",
	PreRun: func(cmd *cobra.Command, args []string) {
		LuetCfg.Viper.BindPFlag("system.database_path", installCmd.Flags().Lookup("system-dbpath"))
		LuetCfg.Viper.BindPFlag("system.rootfs", installCmd.Flags().Lookup("system-target"))
		LuetCfg.Viper.BindPFlag("solver.type", cmd.Flags().Lookup("solver-type"))
		LuetCfg.Viper.BindPFlag("solver.discount", cmd.Flags().Lookup("solver-discount"))
		LuetCfg.Viper.BindPFlag("solver.rate", cmd.Flags().Lookup("solver-rate"))
		LuetCfg.Viper.BindPFlag("solver.max_attempts", cmd.Flags().Lookup("solver-attempts"))
	},
	Long: `Upgrades packages in parallel`,
	Run: func(cmd *cobra.Command, args []string) {
		var systemDB pkg.PackageDatabase

		repos := installer.Repositories{}
		for _, repo := range LuetCfg.SystemRepositories {
			if !repo.Enable {
				continue
			}

			r := installer.NewSystemRepository(repo)
			repos = append(repos, r)
		}

		stype := LuetCfg.Viper.GetString("solver.type")
		discount := LuetCfg.Viper.GetFloat64("solver.discount")
		rate := LuetCfg.Viper.GetFloat64("solver.rate")
		attempts := LuetCfg.Viper.GetInt("solver.max_attempts")

		LuetCfg.GetSolverOptions().Type = stype
		LuetCfg.GetSolverOptions().LearnRate = float32(rate)
		LuetCfg.GetSolverOptions().Discount = float32(discount)
		LuetCfg.GetSolverOptions().MaxAttempts = attempts

		Debug("Solver", LuetCfg.GetSolverOptions().String())

		inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{Concurrency: LuetCfg.GetGeneral().Concurrency, SolverOptions: *LuetCfg.GetSolverOptions()})
		inst.Repositories(repos)
		_, err := inst.SyncRepositories(false)
		if err != nil {
			Fatal("Error: " + err.Error())
		}

		if LuetCfg.GetSystem().DatabaseEngine == "boltdb" {
			systemDB = pkg.NewBoltDatabase(
				filepath.Join(LuetCfg.GetSystem().GetSystemRepoDatabaseDirPath(), "luet.db"))
		} else {
			systemDB = pkg.NewInMemoryDatabase(true)
		}
		system := &installer.System{Database: systemDB, Target: LuetCfg.GetSystem().Rootfs}
		err = inst.Upgrade(system)
		if err != nil {
			Fatal("Error: " + err.Error())
		}
	},
}

func init() {
	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	upgradeCmd.Flags().String("system-dbpath", path, "System db path")
	upgradeCmd.Flags().String("system-target", path, "System rootpath")
	upgradeCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+AvailableResolvers+" )")
	upgradeCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	upgradeCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	upgradeCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	RootCmd.AddCommand(upgradeCmd)
}
