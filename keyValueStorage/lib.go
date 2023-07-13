package keyValueStorage

type assertionArgs struct {
	condition bool
	message   string
}

func assert(condition bool) {
	assertThat(assertionArgs{condition: condition})
}

func assertThat(args assertionArgs) {
	if args.message == "" {
		args.message = "assertion failed"
	}

	if !args.condition {
		panic(args.message)
	}
}
