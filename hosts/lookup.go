package hosts

func AliasLookup(alias string) string {
    // TODO - this is a mock until we get Postgres support for this.
    // This should allow us to do local testing.
    return alias + "/v1.12"
}
