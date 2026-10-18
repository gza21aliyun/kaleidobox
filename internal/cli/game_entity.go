package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newGameEntityCmd(app *CoreApp) *cobra.Command {
	return &cobra.Command{
		Use:   "game_entity <game>",
		Short: "Output the full game entity as JSON (game + works)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			gameQuery := args[0]

			gameID, _, err := resolveGame(w, app, gameQuery)
			if err != nil {
				return err
			}

			gameEntity, err := app.GameService.GetGameEntityByID(gameID)
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(gameEntity, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal game entity: %w", err)
			}

			fmt.Fprintln(w, string(data))
			return nil
		},
	}
}
