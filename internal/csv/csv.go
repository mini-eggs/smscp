package csv

import (
	stdcsv "encoding/csv"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"smscp.xyz/internal/common"
)

type CSV struct{}

func Default() (csv CSV) { return }

func (csv CSV) ToFile(user common.User, notes []common.Note /* msgs []common.Msg */) (*os.File, error) {
	file, err := ioutil.TempFile("", fmt.Sprintf("%s_results.csv", url.QueryEscape(user.Username())))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create csv file")
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	writer := stdcsv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"user_id",
		"user_username",
		"user_phone",
		"note_id",
		"note_text",
	}
	if err := writer.Write(headers); err != nil {
		return nil, errors.Wrap(err, "failed to write header to csv")
	}

	// user
	data := []string{
		user.Token(),
		user.Username(), // TODO: escape for ';'
		user.Phone(),
		"",
		"",
		"",
	}
	if err := writer.Write(data); err != nil {
		return nil, errors.Wrap(err, "failed to write user to csv")
	}

	// notes
	for _, value := range notes {
		data := []string{
			"",
			"",
			"",
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
