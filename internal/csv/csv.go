package csv

import (
	stdcsv "encoding/csv"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"smscp.xyz/internal/common"
)

type CSV struct{}

func Default() (this CSV) { return }

func (this CSV) ToFile(user common.User, notes []common.Note /* msgs []common.Msg */) (*os.File, error) {
	file, err := os.Create("result.csv")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create csv file")
	}
	defer file.Close()

	writer := stdcsv.NewWriter(file)
	defer writer.Flush()

	// user
	data := []string{
		fmt.Sprintf("%d", user.ID()),
		user.Token(),
		user.Username(),
		user.Phone(),
	}
	if err := writer.Write(data); err != nil {
		return nil, errors.Wrap(err, "failed to write user to csv")
	}

	// notes
	for _, value := range notes {
		data := []string{
			fmt.Sprintf("%d", value.ID()),
			value.Token(),
			value.Text(),
		}
		if err := writer.Write(data); err != nil {
			return nil, errors.Wrap(err, "failed to write notes to csv")
		}
	}

	// // msgs
	// for _, value := range msgs {
	// 	data := []string{
	// 		fmt.Sprintf("%d", value.ID()),
	// 		value.Token(),
	// 		value.Text(),
	// 	}
	// 	if err := writer.Write(data); err != nil {
	// 		return nil, errors.Wrap(err, "failed to write messages to csv")
	// 	}
	// }

	return file, nil
}
