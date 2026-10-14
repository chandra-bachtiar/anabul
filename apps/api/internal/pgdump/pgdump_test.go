package pgdump

import "testing"

func TestSiapkanURL(t *testing.T) {
	cases := []struct {
		nama  string
		input string
		want  string
	}{
		{
			nama:  "host lokal diganti",
			input: "postgres://app:app@localhost:5432/finance?sslmode=disable",
			want:  "postgres://app:app@host.docker.internal:5432/finance?sslmode=disable",
		},
		{
			nama:  "ip lokal diganti",
			input: "postgres://app:app@127.0.0.1:5432/finance",
			want:  "postgres://app:app@host.docker.internal:5432/finance",
		},
		{
			nama:  "host jauh dibiarkan",
			input: "postgresql://user:pass@ep-xxx-pooler.ap-southeast-1.aws.neon.tech/anabul?sslmode=require",
			want:  "postgresql://user:pass@ep-xxx-pooler.ap-southeast-1.aws.neon.tech/anabul?sslmode=require",
		},
		{
			nama:  "password berisi kata lokal tidak diubah",
			input: "postgres://user:localhost@db.internal:5432/finance",
			want:  "postgres://user:localhost@db.internal:5432/finance",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			if got := siapkanURL(tc.input); got != tc.want {
				t.Fatalf("siapkanURL = %q, ingin %q", got, tc.want)
			}
		})
	}
}
