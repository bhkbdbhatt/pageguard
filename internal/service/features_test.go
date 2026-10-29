package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/service"
	"github.com/pageguard/pageguard/internal/store/memory"
)

func newService(t *testing.T) (*service.Service, *domain.Project) {
	t.Helper()
	svc := service.New(memory.New(), events.NewHub(), nil)
	project, err := svc.CreateProject(context.Background(), "Acme", "acme")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return svc, project
}

func TestScheduleLifecycle(t *testing.T) {
	svc, project := newService(t)
	ctx := context.Background()

	sched, err := svc.CreateSchedule(ctx, service.ScheduleInput{
		ProjectID: project.ID, Name: "Daily", URL: "https://example.com/",
		Cron: "0 6 * * *", Timezone: "UTC", MaxPages: 10, MaxDepth: 2,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if sched.NextRunAt == nil {
		t.Fatal("expected next run to be scheduled")
	}
	if sched.ScanID.IsEmpty() {
		t.Fatal("expected a backing scan to be created")
	}

	disabled := false
	updated, err := svc.UpdateSchedule(ctx, sched.ID, service.SchedulePatch{Enabled: &disabled})
	if err != nil {
		t.Fatalf("update schedule: %v", err)
	}
	if updated.NextRunAt != nil {
		t.Fatal("expected disabled schedule to have no next run")
	}

	rows, err := svc.Schedules(ctx, project.ID)
	if err != nil {
		t.Fatalf("list schedules: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(rows))
	}
}

func TestCreateScheduleRejectsInvalidCron(t *testing.T) {
	svc, project := newService(t)
	_, err := svc.CreateSchedule(context.Background(), service.ScheduleInput{
		ProjectID: project.ID, Name: "Bad", URL: "https://example.com/", Cron: "not a cron",
	})
	if err == nil {
		t.Fatal("expected invalid cron to be rejected")
	}
}

func TestAlertRuleAndChannel(t *testing.T) {
	svc, project := newService(t)
	ctx := context.Background()

	channel, err := svc.CreateAlertChannel(ctx, service.AlertChannelInput{
		ProjectID: project.ID, Name: "Slack", Kind: "slack", Target: "https://hooks.slack.com/services/x",
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	rule, err := svc.CreateAlertRule(ctx, service.AlertRuleInput{
		ProjectID: project.ID, Name: "Critical", Severity: domain.SevCritical,
		Condition:  domain.AlertCondition{MinSeverity: domain.SevHigh},
		ChannelIDs: []domain.ID{channel.ID},
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if rule.ConditionText == "" {
		t.Fatal("expected a generated condition summary")
	}
	if len(rule.ChannelIDs) != 1 || rule.ChannelIDs[0] != channel.ID {
		t.Fatalf("unexpected channel ids: %v", rule.ChannelIDs)
	}
}

func TestPrivacyProfilesAreSeededAndExtendable(t *testing.T) {
	svc, project := newService(t)
	ctx := context.Background()

	profiles, err := svc.PrivacyProfiles(ctx, project.ID)
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("expected 3 seeded profiles, got %d", len(profiles))
	}

	created, err := svc.CreatePrivacyProfile(ctx, service.PrivacyProfileInput{
		ProjectID: project.ID, Name: "UK", Region: "UK", RegulationIDs: []string{"gdpr"},
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if len(created.Config) == 0 {
		t.Fatal("expected profile config to be generated")
	}

	profiles, err = svc.PrivacyProfiles(ctx, project.ID)
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) != 4 {
		t.Fatalf("expected 4 profiles, got %d", len(profiles))
	}
}

func TestJourneyCreationAndListing(t *testing.T) {
	svc, project := newService(t)
	ctx := context.Background()

	journey, err := svc.CreateJourney(ctx, service.JourneyInput{
		ProjectID: project.ID, Name: "Checkout", BaseURL: "https://example.com/",
		Steps: []domain.JourneyStep{
			{Order: 1, Action: "navigate", Value: "https://example.com/"},
			{Order: 2, Action: "click", Selector: "#buy"},
		},
	})
	if err != nil {
		t.Fatalf("create journey: %v", err)
	}
	if len(journey.Definition) == 0 {
		t.Fatal("expected a canonical definition")
	}

	rows, err := svc.Journeys(ctx, project.ID)
	if err != nil {
		t.Fatalf("list journeys: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != journey.ID {
		t.Fatalf("unexpected journeys: %+v", rows)
	}

	_, err = svc.CreateJourney(ctx, service.JourneyInput{
		ProjectID: project.ID, Name: "Broken",
		Steps: []domain.JourneyStep{{Order: 1, Action: "assert"}},
	})
	if err == nil {
		t.Fatal("expected unsupported action to be rejected")
	}
}

func TestGenerateReportsWithoutRuns(t *testing.T) {
	svc, project := newService(t)
	ctx := context.Background()

	for _, templateID := range []string{"tpl-governance", "tpl-consent", "tpl-inventory", "tpl-journey", "tpl-privacy"} {
		report, err := svc.GenerateReport(ctx, service.ReportInput{
			ProjectID: project.ID, TemplateID: templateID, Format: "json",
		})
		if err != nil {
			t.Fatalf("generate %s: %v", templateID, err)
		}
		if report.SizeBytes == 0 {
			t.Fatalf("%s report is empty", templateID)
		}
	}

	rows, err := svc.Reports(ctx, project.ID, 50)
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5 reports, got %d", len(rows))
	}
}

func TestReportScheduleLifecycle(t *testing.T) {
	svc, project := newService(t)
	ctx := context.Background()

	sched, err := svc.CreateReportSchedule(ctx, service.ReportScheduleInput{
		ProjectID: project.ID, TemplateID: "tpl-governance", Name: "Weekly",
		Cron: "0 6 * * 1", Timezone: "UTC", Format: "csv",
	})
	if err != nil {
		t.Fatalf("create report schedule: %v", err)
	}
	if sched.NextRunAt == nil {
		t.Fatal("expected next run to be scheduled")
	}

	due := time.Now().UTC().Add(8 * 24 * time.Hour)
	svc.ProcessReportSchedules(ctx, due)
	rows, err := svc.Reports(ctx, project.ID, 10)
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected scheduled report, got %d", len(rows))
	}
}

func TestInventoryEmptyProject(t *testing.T) {
	svc, project := newService(t)
	snapshot, err := svc.Inventory(context.Background(), project.ID, "")
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if snapshot.ProjectID != project.ID {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}
