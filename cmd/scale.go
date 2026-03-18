package cmd

import (
	"fmt"
	"strconv"

	"github.com/abix-/k3sc/internal/k8s"
	"github.com/abix-/k3sc/internal/types"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(scaleCmd)
}

var scaleCmd = &cobra.Command{
	Use:   "scale [N]",
	Short: "Get or set max concurrent agent slots",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runScale,
}

func runScale(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cs, err := k8s.NewClient()
	if err != nil {
		return err
	}

	deploy, err := cs.AppsV1().Deployments(types.Namespace).Get(ctx, "k3sc-operator", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get operator deployment: %w", err)
	}

	container := &deploy.Spec.Template.Spec.Containers[0]

	// read current value
	current := 5
	for _, e := range container.Env {
		if e.Name == "MAX_SLOTS" {
			if n, err := strconv.Atoi(e.Value); err == nil {
				current = n
			}
		}
	}

	if len(args) == 0 {
		fmt.Printf("max slots: %d\n", current)
		return nil
	}

	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 || n > 26 {
		return fmt.Errorf("slots must be 1-26, got: %s", args[0])
	}

	// update or add MAX_SLOTS env var
	found := false
	for i := range container.Env {
		if container.Env[i].Name == "MAX_SLOTS" {
			container.Env[i].Value = strconv.Itoa(n)
			found = true
			break
		}
	}
	if !found {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  "MAX_SLOTS",
			Value: strconv.Itoa(n),
		})
	}

	if _, err := cs.AppsV1().Deployments(types.Namespace).Update(ctx, deploy, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}

	fmt.Printf("max slots: %d -> %d (operator restarting)\n", current, n)
	return nil
}
