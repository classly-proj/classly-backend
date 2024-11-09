package util

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lpernett/godotenv"
)

var Config struct {
	Database struct {
		FileName  string
		QueueSize int
	}

	Server struct {
		Port int
	}

	General struct {
		UpdateCourses bool
	}
}

func LoadEnvFile() {
	err := godotenv.Load()

	if err != nil {
		Log.Error("Error loading .env file! Would you like to create one now? (y/N):")

		var response string

		for {
			_, err := fmt.Scanln(&response)

			if err != nil {
				Log.Error("Error reading response")
				os.Exit(1)
				return
			}

			if response == "y" || response == "Y" {
				Log.Basic("Creating .env file")

				file, err := os.Create(".env")

				if err != nil {
					Log.Error("Error creating .env file")
					os.Exit(1)
					return
				}

				file.WriteString("DATABASE_FILE_NAME=\n")
				file.WriteString("DATABASE_QUEUE_SIZE=\n")
				file.WriteString("SERVER_PORT=\n")
				file.WriteString("GENERAL_UPDATE_COURSES=\n")

				file.Close()

				Log.Basic("Created .env file")

				break
			} else if response == "n" || response == "N" {
				Log.Error("Exiting")
				os.Exit(1)
				return
			} else {
				Log.Error("Invalid response")
			}
		}
	}

	Log.Basic("Loaded .env file")

	var tmp any

	if tmp = os.Getenv("DATABASE_FILE_NAME"); tmp == "" {
		Log.Error("DATABASE_FILE_NAME not set (string)")
		os.Exit(1)
	} else {
		Config.Database.FileName = tmp.(string)
	}

	if tmp = os.Getenv("DATABASE_QUEUE_SIZE"); tmp == "" {
		Log.Error("DATABASE_QUEUE_SIZE not set (int)")
		os.Exit(1)
	} else {
		if i, err := strconv.ParseInt(tmp.(string), 10, 64); err != nil {
			Log.Error("DATABASE_QUEUE_SIZE not an integer")
			os.Exit(1)
		} else {
			Config.Database.QueueSize = int(i)
		}
	}

	if tmp = os.Getenv("SERVER_PORT"); tmp == "" {
		Log.Error("SERVER_PORT not set (int)")
		os.Exit(1)
	} else {
		if i, err := strconv.ParseInt(tmp.(string), 10, 64); err != nil {
			Log.Error("SERVER_PORT not an integer")
			os.Exit(1)
		} else {
			Config.Server.Port = int(i)
		}
	}

	if tmp = os.Getenv("GENERAL_UPDATE_COURSES"); tmp == "" {
		Log.Error("GENERAL_UPDATE_COURSES not set (bool/truthy)")
		os.Exit(1)
	} else {
		if b, err := strconv.ParseBool(tmp.(string)); err != nil {
			Log.Error("GENERAL_UPDATE_COURSES not a truthy value")
			os.Exit(1)
		} else {
			Config.General.UpdateCourses = b
		}
	}

	Log.Status("Loaded environment variables")
}
