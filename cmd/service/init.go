package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openfresh/ecs-formation/client"
	cmdutil "github.com/openfresh/ecs-formation/cmd/util"
	"github.com/openfresh/ecs-formation/service"
	"github.com/openfresh/ecs-formation/service/types"
	"github.com/openfresh/ecs-formation/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	projectDir  string
	cluster     string
	serviceName string
	parameters  map[string]string
	jsonOutput  bool
)

var ServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Amazon ECS Service",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		pd, err := cmdutil.GetProjectDir()
		if err != nil {
			return err
		}
		projectDir = pd

		region := viper.GetString("aws_region")
		client.Init(region, false)

		cl, err := cmd.Flags().GetString("cluster")
		if err != nil {
			return err
		}
		if cl == "" {
			return errors.New("-c (--cluster) is required")
		}

		cluster = cl

		sv, err := cmd.Flags().GetString("service")
		if err != nil {
			return err
		}
		serviceName = sv

		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		if serviceName == "" && all == false {
			return errors.New("should specify '-s service_name' or '--all' option")
		}

		paramTokens, err := cmd.Flags().GetStringSlice("parameter")
		if err != nil {
			return err
		}
		parameters = util.ParseKeyValues(paramTokens)

		jo, err := cmd.Flags().GetBool("json-output")
		if err != nil {
			return err
		}
		jsonOutput = jo

		return nil
	},
}

func init() {
	ServiceCmd.AddCommand(planCmd)
	ServiceCmd.AddCommand(applyCmd)

	ServiceCmd.PersistentFlags().StringP("cluster", "c", "", "ECS Cluster")
	ServiceCmd.PersistentFlags().StringP("service", "s", "", "ECS Service")
	ServiceCmd.PersistentFlags().StringSliceP("parameter", "p", make([]string, 0), "parameter 'key=value'")
	ServiceCmd.PersistentFlags().BoolP("json-output", "j", false, "Print json format")
}

func createClusterPlans(srv service.ClusterService) ([]*types.ServiceUpdatePlan, error) {

	if jsonOutput {
		util.Output = false
		defer func() {
			util.Output = true
		}()
	}

	util.PrintlnYellow("Checking services on clusters...")
	plans, err := srv.CreateServiceUpdatePlans()
	if err != nil {
		return make([]*types.ServiceUpdatePlan, 0), err
	}

	for _, plan := range plans {
		util.PrintlnYellow("Current status of ECS Cluster '%s':", plan.Name)
		if len(plan.InstanceARNs) > 0 {
			util.PrintlnYellow("    Container Instances as follows:")
			for _, instance := range plan.InstanceARNs {
				util.PrintlnYellow("        %s:", *instance)
			}
		}

		util.Println()
		util.PrintlnYellow("    Services as follows:")
		if len(plan.CurrentServices) == 0 {
			util.PrintlnYellow("         No services are deployed.")
		}

		for _, cst := range plan.CurrentServices {
			cs := cst.Service
			util.PrintlnYellow("        ####[%s]####\n", *cs.ServiceName)
			util.PrintlnYellow("        ServiceARN = %s", *cs.ServiceArn)
			util.PrintlnYellow("        TaskDefinition = %s", *cs.TaskDefinition)
			util.PrintlnYellow("        DesiredCount = %d", *cs.DesiredCount)
			util.PrintlnYellow("        PendingCount = %d", *cs.PendingCount)
			util.PrintlnYellow("        RunningCount = %d", *cs.RunningCount)
			if cs.RoleArn != nil {
				util.PrintlnYellow("        Role = %d", *cs.RoleArn)
			}
			if cs.DeploymentConfiguration != nil {
				util.PrintlnYellow("        MinimumHealthyPercent = %d", *cs.DeploymentConfiguration.MinimumHealthyPercent)
				util.PrintlnYellow("        MaximumPercent = %d", *cs.DeploymentConfiguration.MaximumPercent)
			}
			for _, lb := range cs.LoadBalancers {
				if lb.LoadBalancerName != nil {
					util.PrintlnYellow("        ELB = %s:", *lb.LoadBalancerName)
				}
				if lb.TargetGroupArn != nil {
					util.PrintlnYellow("        TargetGroupARN = %s:", *lb.TargetGroupArn)
				}
				util.PrintlnYellow("            ContainerName = %v", *lb.ContainerName)
				util.PrintlnYellow("            ContainerPort = %v", *lb.ContainerPort)
			}
			util.PrintlnYellow("        STATUS = %s", *cs.Status)

			if cst.AutoScaling != nil {
				asg := cst.AutoScaling
				util.PrintlnYellow("        AutoScaling:")
				util.PrintlnYellow("            ResourceId = %s", *asg.ResourceId)
				util.PrintlnYellow("            MinCapacity = %v", *asg.MinCapacity)
				util.PrintlnYellow("            MaxCapacity = %v", *asg.MaxCapacity)
				util.PrintlnYellow("            RoleARN = %s", *asg.RoleARN)
			}

			if len(cs.PlacementStrategy) > 0 {
				util.PrintlnYellow("        PlacementStrategy:")
			}
			for _, ps := range cs.PlacementStrategy {
				util.PrintlnYellow("          -")
				util.PrintlnYellow("            Type = %s", *ps.Type)
				util.PrintlnYellow("            Field = %s", *ps.Field)
			}

			if len(cs.PlacementConstraints) > 0 {
				util.PrintlnYellow("        PlacementConstraints:")
			}
			for _, pc := range cs.PlacementConstraints {
				util.PrintlnYellow("          -")
				util.PrintlnYellow("            Type = %s", *pc.Type)
				util.PrintlnYellow("            Field = %s", *pc.Expression)
			}

			util.Println()
		}

		util.Println()
		util.PrintlnYellow("Service update plan '%s':", plan.Name)

		util.PrintlnYellow("    Services:")
		for _, add := range plan.NewServices {
			util.PrintlnYellow("        ####[%s]####\n", add.Name)
			util.PrintlnYellow("        TaskDefinition = %s", add.TaskDefinition)
			util.PrintlnYellow("        DesiredCount = %d", add.DesiredCount)
			util.PrintlnYellow("        KeepDesiredCount = %t", add.KeepDesiredCount)
			if add.MinimumHealthyPercent.Valid {
				util.PrintlnYellow("        MinimumHealthyPercent = %d", add.MinimumHealthyPercent.Int64)
			}
			if add.MaximumPercent.Valid {
				util.PrintlnYellow("        MaximumPercent = %d", add.MaximumPercent.Int64)
			}
			util.PrintlnYellow("        Role = %v", add.Role)
			for _, lb := range add.LoadBalancers {
				if lb.Name.Valid {
					util.PrintlnYellow("        ELB:%v", lb.Name.String)
				}
				if lb.TargetGroupARN.Valid {
					util.PrintlnYellow("            TargetGroupARN:%v", lb.TargetGroupARN.String)
				}
				util.PrintlnYellow("            ContainerName:%v", lb.ContainerName)
				util.PrintlnYellow("            ContainerPort:%v", lb.ContainerPort)
			}

			if add.AutoScaling != nil && add.AutoScaling.Target != nil {
				asg := add.AutoScaling.Target
				util.PrintlnYellow("        AutoScaling:")
				util.PrintlnYellow("            MinCapacity = %v", asg.MinCapacity)
				util.PrintlnYellow("            MaxCapacity = %v", asg.MaxCapacity)
				util.PrintlnYellow("            RoleARN = %s", asg.Role)
			}

			if len(add.PlacementStrategy) > 0 {
				for _, ps := range add.PlacementStrategy {
					util.PrintlnYellow("        PlacementStrategy:")
					util.PrintlnYellow("          -")
					util.PrintlnYellow("            Type = %v", ps.Type)
					util.PrintlnYellow("            Field = %v", ps.Field)
				}
			}

			if len(add.PlacementConstraints) > 0 {
				for _, pc := range add.PlacementConstraints {
					util.PrintlnYellow("        PlacementConstraints:")
					util.PrintlnYellow("          -")
					util.PrintlnYellow("            Type = %v", pc.Type)
					util.PrintlnYellow("            Expression = %v", pc.Expression)
				}
			}
			util.Println()
		}

		util.Println()
	}

	if jsonOutput {
		bt, err := json.Marshal(&plans)
		if err != nil {
			return plans, err
		}
		fmt.Println(string(bt))
	}

	return plans, nil
}
