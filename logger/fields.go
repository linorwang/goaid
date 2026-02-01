package logger

import "time"

func String(key, val string) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Int(key string, val int) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Int64(key string, val int64) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Int32(key string, val int32) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Float64(key string, val float64) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Bool(key string, val bool) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Time(key string, val time.Time) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Duration(key string, val time.Duration) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Strs(key string, vals []string) Field {
	return Field{
		Key:   key,
		Value: vals,
	}
}

func Err(err error) Field {
	return Field{
		Key:   "error",
		Value: err,
	}
}

func Any(key string, val any) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func Struct(key string, val any) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}
