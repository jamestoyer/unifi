package utils

func Int32PtrValue(val *int) *int32 {
	if val == nil {
		return nil
	}

	i := int32(*val)
	return &i
}

func IntPtrValue(val *int32) *int {
	if val == nil {
		return nil
	}

	i := int(*val)
	return &i
}

func StringPtr(val string) *string {
	return &val
}
