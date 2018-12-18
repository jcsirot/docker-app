package commands

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type upgradeOptions struct {
	parametersOptions
	credentialOptions
	registryOptions
	pullOptions
	bundleOrDockerApp string
	sendRegistryAuth  bool
}

func upgradeCmd(dockerCli command.Cli) *cobra.Command {
	var opts upgradeOptions
	cmd := &cobra.Command{
		Use:   "upgrade <installation-name> [options]",
		Short: "Upgrade an installed application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(dockerCli, args[0], opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	opts.registryOptions.addFlags(cmd.Flags())
	opts.pullOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.bundleOrDockerApp, "bundle", "", "Override with new bundle or Docker App")
	cmd.Flags().BoolVar(&opts.sendRegistryAuth, "with-registry-auth", false, "Sends registry auth")

	return cmd
}

func runUpgrade(dockerCli command.Cli, installationName string, opts upgradeOptions) error {
	defer muteDockerCli(dockerCli)()
	targetContext := getTargetContext(opts.targetContext, dockerCli.CurrentContext())
	h := duffleHome()
	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	c, err := claimStore.Read(installationName)
	if err != nil {
		return err
	}

	if opts.bundleOrDockerApp != "" {
		b, err := resolveBundle(dockerCli, opts.bundleOrDockerApp, opts.pull, opts.insecureRegistries)
		if err != nil {
			return err
		}
		c.Bundle = b
	}
	c.Parameters, err = mergeBundleParameters(c.Bundle,
		withFileParameters(opts.parametersFiles),
		withCommandLineParameters(opts.overrides),
		withSendRegistryAuth(opts.sendRegistryAuth),
	)
	if err != nil {
		return err
	}

	bind, err := requiredClaimBindMount(c, targetContext, dockerCli)
	if err != nil {
		return err
	}
	driverImpl, errBuf, err := prepareDriver(dockerCli, bind, nil)
	if err != nil {
		return err
	}
	creds, err := prepareCredentialSet(c.Bundle,
		addNamedCredentialSets(opts.credentialsets),
		addDockerCredentials(targetContext, dockerCli.ContextStore()),
		addRegistryCredentials(c.Parameters, dockerCli))
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, c.Bundle.Credentials); err != nil {
		return err
	}
	u := &action.Upgrade{
		Driver: driverImpl,
	}
	err = u.Run(&c, creds, dockerCli.Out())
	err2 := claimStore.Store(c)
	if err != nil {
		return fmt.Errorf("upgrade failed: %s", errBuf)
	}
	return err2
}
