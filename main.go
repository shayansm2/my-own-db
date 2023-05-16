package main

import (
	"fmt"
	"myOwnDb/filePersistence"
)

func main() {
	//testFilePersistence()
	testLogging()
}

func testLogging() {
	fp, err := filePersistence.LogCreate("tmp.log")
	if err != nil {
		fmt.Println(err)
		return
	}

	log := "this is the first line of the log"

	err = filePersistence.LogAppend(fp, log)
	if err != nil {
		fmt.Println(err)
		return
	}

	log = "this is the second line of the log"

	err = filePersistence.LogAppend(fp, log)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func testFilePersistence() {
	err := filePersistence.SaveData("temp.file", []byte{1, 2, 3, 4})
	if err != nil {
		return
	} else {
		fmt.Println(err)
	}
}
