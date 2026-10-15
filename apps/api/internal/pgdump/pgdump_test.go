package pgdump

import "testing"

func TestPrepareURL(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "local host is rewritten",
			input: "postgres://app:app@localhost:5432/finance?sslmode=disable",
			want:  "postgres://app:app@host.docker.internal:5432/finance?sslmode=disable",
		},
		{
			name:  "local ip is rewritten",
			input: "postgres://app:app@127.0.0.1:5432/finance",
			want:  "postgres://app:app@host.docker.internal:5432/finance",
		},
		{
			name:  "remote host is left alone",
			input: "postgresql://user:pass@ep-xxx-pooler.ap-southeast-1.aws.neon.tech/anabul?sslmode=require",
			want:  "postgresql://user:pass@ep-xxx-pooler.ap-southeast-1.aws.neon.tech/anabul?sslmode=require",
		},
		{
			name:  "localhost inside the password is not rewritten",
			input: "postgres://user:localhost@db.internal:5432/finance",
			want:  "postgres://user:localhost@db.internal:5432/finance",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := prepareURL(tc.input); got != tc.want {
				t.Fatalf("prepareURL = %q, want %q", got, tc.want)
			}
		})
	}
}
