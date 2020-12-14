package lib

// Fragment - Struct for storing metadata about range request response data
type Fragment struct {
	Index int // Store index of data range inside the original request
	Data []byte
}
