package sendsms

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testProvider struct {
	name     string
	sendFunc func(context.Context, *SMSRequest) (*SMSResponse, error)
	calls    int
	lastReq  *SMSRequest
}

func (p *testProvider) Send(ctx context.Context, req *SMSRequest) (*SMSResponse, error) {
	p.calls++
	copied := *req
	p.lastReq = &copied
	if p.sendFunc != nil {
		return p.sendFunc(ctx, req)
	}
	return &SMSResponse{
		Success:   true,
		MessageID: p.name + "-message",
		Provider:  p.name,
	}, nil
}

func (p *testProvider) SendBatch(ctx context.Context, reqs []*SMSRequest) ([]*SMSResponse, error) {
	responses := make([]*SMSResponse, len(reqs))
	for i, req := range reqs {
		resp, err := p.Send(ctx, req)
		if err != nil {
			return responses, err
		}
		responses[i] = resp
	}
	return responses, nil
}

func (p *testProvider) Name() string {
	return p.name
}

func (p *testProvider) ValidateConfig() error {
	return nil
}

func (p *testProvider) GetBalance(ctx context.Context) (*Balance, error) {
	return &Balance{Amount: 1, Currency: "CNY", UpdatedAt: time.Now()}, nil
}

func (p *testProvider) HealthCheck(ctx context.Context) bool {
	return true
}

func (p *testProvider) GetErrorType(err error) ErrorType {
	return ErrorTypeProvider
}

func (p *testProvider) IsRetryable(err error) bool {
	return true
}

func TestNewUsesSingleProviderDefaultsAndNoRedis(t *testing.T) {
	provider := &testProvider{name: "mock"}

	client, err := New(
		WithProvider("mock", provider),
		WithFailover(false),
		WithDefaultTemplate("TPL_1"),
		WithDefaultSign("SIGN"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	original := &SMSRequest{Phone: "13800138000"}
	resp, err := client.Send(context.Background(), original)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !resp.Success {
		t.Fatalf("Send() success = false")
	}
	if resp.Provider != "mock" {
		t.Fatalf("Provider = %q, want mock", resp.Provider)
	}
	if provider.lastReq.Template != "TPL_1" {
		t.Fatalf("Template = %q, want TPL_1", provider.lastReq.Template)
	}
	if provider.lastReq.SignName != "SIGN" {
		t.Fatalf("SignName = %q, want SIGN", provider.lastReq.SignName)
	}
	if original.Template != "" || original.SignName != "" {
		t.Fatalf("Send mutated original request: %+v", original)
	}
}

func TestFailoverTriesAllBackups(t *testing.T) {
	fail := func(name string) *testProvider {
		return &testProvider{
			name: name,
			sendFunc: func(context.Context, *SMSRequest) (*SMSResponse, error) {
				return &SMSResponse{Success: false, Provider: name}, errors.New("send failed")
			},
		}
	}

	primary := fail("primary")
	backup1 := fail("backup1")
	backup2 := &testProvider{name: "backup2"}

	config := DefaultConfig()
	config.RetryTimes = 0
	config.EnableFailover = true

	client, err := NewSMSClient("primary", []string{"backup1", "backup2"}, map[string]SMSProvider{
		"primary": primary,
		"backup1": backup1,
		"backup2": backup2,
	}, nil, config)
	if err != nil {
		t.Fatalf("NewSMSClient() error = %v", err)
	}

	resp, err := client.Send(context.Background(), &SMSRequest{
		Phone:    "13800138000",
		Template: "TPL_1",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if resp.Provider != "backup2" {
		t.Fatalf("Provider = %q, want backup2", resp.Provider)
	}
	if primary.calls != 1 || backup1.calls != 1 || backup2.calls != 1 {
		t.Fatalf("calls = primary:%d backup1:%d backup2:%d", primary.calls, backup1.calls, backup2.calls)
	}
}

func TestSendAppliesTimeout(t *testing.T) {
	provider := &testProvider{
		name: "slow",
		sendFunc: func(ctx context.Context, req *SMSRequest) (*SMSResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	config := DefaultConfig()
	config.EnableFailover = false
	config.RetryTimes = 0
	config.Timeout = 10 * time.Millisecond

	client, err := NewSMSClient("slow", nil, map[string]SMSProvider{"slow": provider}, nil, config)
	if err != nil {
		t.Fatalf("NewSMSClient() error = %v", err)
	}

	start := time.Now()
	_, err = client.Send(context.Background(), &SMSRequest{
		Phone:    "13800138000",
		Template: "TPL_1",
	})
	if err == nil {
		t.Fatal("Send() error = nil, want deadline error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Send() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("Send() took %s, want timeout to apply quickly", elapsed)
	}
}
