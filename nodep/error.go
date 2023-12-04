package nodep

func WrapError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
