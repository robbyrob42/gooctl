package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dorkitude/gooctl/internal/auth"
	gmailpkg "github.com/dorkitude/gooctl/internal/gmail"
	"github.com/dorkitude/gooctl/internal/ui"
	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Search and manage emails",
	Long:  `Commands for searching, reading, and sending emails.`,
}

var mailSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search emails",
	Long: `Search emails using Gmail query syntax.

Examples:
  gooctl mail search "from:boss@example.com"
  gooctl mail search "subject:invoice"
  gooctl mail search "is:unread"
  gooctl mail search "after:2024/01/01 before:2024/02/01"
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		query := strings.Join(args, " ")

		limit, _ := cmd.Flags().GetInt64("limit")

		client, err := getGmailClient(ctx)
		if err != nil {
			return err
		}

		messages, err := client.Search(ctx, query, limit)
		if err != nil {
			return err
		}

		if len(messages) == 0 {
			fmt.Println(ui.Warning_("No messages found"))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📧 Found %d messages", len(messages))))
		fmt.Println()

		for _, msg := range messages {
			printMailSummary(msg)
		}

		return nil
	},
}

var mailReadCmd = &cobra.Command{
	Use:   "read [message-id]",
	Short: "Read a specific email",
	Long:  `Display the full content of an email by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		messageID := args[0]

		client, err := getGmailClient(ctx)
		if err != nil {
			return err
		}

		msg, err := client.GetMessage(ctx, messageID)
		if err != nil {
			return err
		}

		printMailFull(msg)
		return nil
	},
}

var mailThreadCmd = &cobra.Command{
	Use:   "thread [thread-id]",
	Short: "View an email thread",
	Long:  `Display all messages in a thread.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		threadID := args[0]

		client, err := getGmailClient(ctx)
		if err != nil {
			return err
		}

		messages, err := client.GetThread(ctx, threadID)
		if err != nil {
			return err
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📧 Thread with %d messages", len(messages))))
		fmt.Println()

		for i, msg := range messages {
			fmt.Println(ui.SubtleStyle.Render(fmt.Sprintf("--- Message %d of %d ---", i+1, len(messages))))
			printMailFull(msg)
			fmt.Println()
		}

		return nil
	},
}

var mailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a new email",
	Long:  `Send a new email. Requires --to, --subject, and --body flags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		to, _ := cmd.Flags().GetString("to")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")

		if to == "" || subject == "" || body == "" {
			return fmt.Errorf("--to, --subject, and --body are required")
		}

		client, err := getGmailClient(ctx)
		if err != nil {
			return err
		}

		msg, err := client.Send(ctx, to, subject, body)
		if err != nil {
			return err
		}

		fmt.Println(ui.Success("Email sent!"))
		fmt.Println(ui.SubtleStyle.Render("Message ID: " + msg.ID))
		return nil
	},
}

var mailReplyCmd = &cobra.Command{
	Use:   "reply [message-id]",
	Short: "Reply to an email",
	Long:  `Reply to an existing email in its thread.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		messageID := args[0]

		body, _ := cmd.Flags().GetString("body")
		if body == "" {
			return fmt.Errorf("--body is required")
		}

		client, err := getGmailClient(ctx)
		if err != nil {
			return err
		}

		// Get original message to find thread ID
		original, err := client.GetMessage(ctx, messageID)
		if err != nil {
			return err
		}

		msg, err := client.Reply(ctx, original.ThreadID, messageID, body)
		if err != nil {
			return err
		}

		fmt.Println(ui.Success("Reply sent!"))
		fmt.Println(ui.SubtleStyle.Render("Message ID: " + msg.ID))
		return nil
	},
}

var mailUnreadCmd = &cobra.Command{
	Use:   "unread",
	Short: "List unread emails",
	Long:  `Show all unread emails in your inbox.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		limit, _ := cmd.Flags().GetInt64("limit")

		client, err := getGmailClient(ctx)
		if err != nil {
			return err
		}

		messages, err := client.Search(ctx, "is:unread", limit)
		if err != nil {
			return err
		}

		if len(messages) == 0 {
			fmt.Println(ui.Success("No unread messages! 🎉"))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📬 %d unread messages", len(messages))))
		fmt.Println()

		for _, msg := range messages {
			printMailSummary(msg)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mailCmd)

	mailCmd.AddCommand(mailSearchCmd)
	mailSearchCmd.Flags().Int64P("limit", "n", 10, "Maximum number of results")

	mailCmd.AddCommand(mailReadCmd)
	mailCmd.AddCommand(mailThreadCmd)

	mailCmd.AddCommand(mailSendCmd)
	mailSendCmd.Flags().StringP("to", "t", "", "Recipient email address")
	mailSendCmd.Flags().StringP("subject", "s", "", "Email subject")
	mailSendCmd.Flags().StringP("body", "b", "", "Email body")

	mailCmd.AddCommand(mailReplyCmd)
	mailReplyCmd.Flags().StringP("body", "b", "", "Reply body")

	mailCmd.AddCommand(mailUnreadCmd)
	mailUnreadCmd.Flags().Int64P("limit", "n", 10, "Maximum number of results")
}

func getGmailClient(ctx context.Context) (*gmailpkg.Client, error) {
	config, err := auth.NewConfig()
	if err != nil {
		return nil, err
	}

	httpClient, err := config.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	return gmailpkg.NewClient(ctx, httpClient)
}

func printMailSummary(msg *gmailpkg.Message) {
	unreadMarker := " "
	if msg.IsUnread {
		unreadMarker = ui.WarningStyle.Render("●")
	}

	fmt.Printf("%s %s %s\n",
		unreadMarker,
		ui.LabelStyle.Render(truncate(msg.From, 25)),
		msg.Subject,
	)
	fmt.Printf("  %s\n",
		ui.SubtleStyle.Render(fmt.Sprintf("ID: %s | %s", msg.ID, msg.Date.Format("Jan 2, 2006 15:04"))),
	)
	if msg.Snippet != "" {
		fmt.Printf("  %s\n", ui.SubtleStyle.Render(truncate(msg.Snippet, 80)))
	}
	fmt.Println()
}

func printMailFull(msg *gmailpkg.Message) {
	fmt.Println(ui.BoxStyle.Render(
		ui.LabelStyle.Render("Subject: ") + msg.Subject + "\n" +
			ui.LabelStyle.Render("From:    ") + msg.From + "\n" +
			ui.LabelStyle.Render("To:      ") + msg.To + "\n" +
			ui.LabelStyle.Render("Date:    ") + msg.Date.Format("Mon, Jan 2, 2006 3:04 PM") + "\n" +
			ui.LabelStyle.Render("ID:      ") + msg.ID,
	))
	fmt.Println()
	if msg.Body != "" {
		fmt.Println(msg.Body)
	} else {
		fmt.Println(ui.SubtleStyle.Render("(No plain text body available)"))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
