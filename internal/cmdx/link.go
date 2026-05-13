package cmdx

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/model"
	"github.com/Olyxz16/tkt/internal/render"
)

var (
	linkTypeFlag string
)

var linkCmd = &cobra.Command{
	Use:   "link <issue-id> <target-id>",
	Short: "Link two issues",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		if linkTypeFlag == "" {
			fmt.Fprint(os.Stderr, "Error: link type is required (--type flag)\n")
			os.Exit(1)
		}

		if err := svc.AddLink(cmd.Context(), args[0], args[1], linkTypeFlag); err != nil {
			handleError(err)
		}
		if quietFlag {
			fmt.Println(args[0])
			return
		}
		if getOutputMode() == render.OutputJSON {
			render.JSON(map[string]interface{}{"source": args[0], "target": args[1], "type": linkTypeFlag, "linked": true})
			return
		}
		fmt.Println("Linked successfully")
	},
}

var (
	logTypeFlag string
	logDateFlag string
)

var logCmd = &cobra.Command{
	Use:   "log <issue-id> <duration>",
	Short: "Log work time for an issue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		svc, _, err := buildService()
		if err != nil {
			handleError(err)
		}

		issueID := args[0]
		durationStr := args[1]
		minutes, err := parseDuration(durationStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid duration: %s\n", err)
			os.Exit(1)
		}

		date := time.Now()
		if logDateFlag != "" {
			date, err = time.Parse("2006-01-02", logDateFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid date format (YYYY-MM-DD): %s\n", err)
				os.Exit(1)
			}
		}

		item := model.WorkItem{
			Date: date.UnixMilli(),
			Duration: &model.Duration{
				Minutes:      minutes,
				Presentation: durationStr,
			},
		}
		if logTypeFlag != "" {
			item.Type = &model.WorkItemType{Name: logTypeFlag}
		}

		created, err := svc.AddWorkItem(cmd.Context(), issueID, item)
		if err != nil {
			handleError(err)
		}

		if quietFlag {
			fmt.Println(created.ID)
			return
		}

		if getOutputMode() == render.OutputJSON {
			render.JSON(created)
			return
		}

		fmt.Printf("Logged %s on %s\n", created.Duration.Presentation, issueID)
	},
}

func parseDuration(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	var total int
	var num strings.Builder
	for _, r := range s {
		switch r {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			num.WriteRune(r)
		case 'd':
			n, _ := strconv.Atoi(num.String())
			total += n * 8 * 60 // assume 8h workday
			num.Reset()
		case 'h':
			n, _ := strconv.Atoi(num.String())
			total += n * 60
			num.Reset()
		case 'm':
			n, _ := strconv.Atoi(num.String())
			total += n
			num.Reset()
		case ' ':
			// ignore spaces
		default:
			return 0, fmt.Errorf("invalid character: %c", r)
		}
	}
	if num.Len() > 0 {
		n, _ := strconv.Atoi(num.String())
		total += n // assume minutes if no unit
	}
	return total, nil
}

func init() {
	linkCmd.Flags().StringVarP(&linkTypeFlag, "type", "t", "", "Link type ID")

	logCmd.Flags().StringVarP(&logTypeFlag, "type", "t", "", "Work item type")
	logCmd.Flags().StringVarP(&logDateFlag, "date", "d", "", "Date (YYYY-MM-DD)")

	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(logCmd)
}
