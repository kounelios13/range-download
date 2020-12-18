package lib

// Fragment - Struct for storing metadata about range request response data
type Fragment struct {
	Index int64 // Store index of data range inside the original request
	Data []byte
}
