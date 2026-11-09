package main

func main() {
	getCreator("linux")
}

func getCreator(os string) string {
	var creator string
	switch os {
	case "linux":
		creator = "linux torvald"
	case "windows":
		creator = "bill gates"
	case "mac":
		creator = "A steve"
	default:
		creator = "unknown"
	}
	return creator
}
