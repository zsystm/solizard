package main

const SelectableListSize = 4

func shouldSupportSearchMode(listLen int) bool {
	return listLen > SelectableListSize
}
