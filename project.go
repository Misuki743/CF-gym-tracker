package main

import (
  "encoding/json"
  "fmt"
  "github.com/rodaine/table"
  "github.com/savioxavier/termlink"
  "io"
  "net/http"
  "slices"
  "sort"
  "strconv"
  "time"
)

func runSpinner(message string, interrupt chan bool) {
  for {
    select {
    case <- interrupt:
      return
    default:
      for _, ch := range "-\\|/" {
        fmt.Printf("\r%s %c", message, ch)
        time.Sleep(100 * time.Millisecond)
      }
    }
  }
}

type SubmissionObject struct {
  Status string
  Result []Submission
}

type Submission struct {
  Id int
  ContestId int
  Problem Problem 
  Verdict string
}

type Problem struct {
  ContestId int
  Name string
  Index string
}

func fetchSubmission(handle string) []Submission {
  interrupt := make(chan bool)
  go runSpinner("fetching submissions", interrupt)

  response, _ := http.Get("https://codeforces.com/api/user.status?handle=" + handle + "&from=1&count=10000")
  body, _ := io.ReadAll(response.Body)

  interrupt <- true

  var fetchedObject SubmissionObject
  json.Unmarshal(body, &fetchedObject)

  if len(fetchedObject.Result) == 0 {
    fmt.Println("\rfetch submissions failed!   ")
  } else {
    fmt.Println("\rdone!                       ")
  }

  return fetchedObject.Result
}

type ContestObject struct {
  Status string
  Result []Contest
}

type Contest struct {
  Id int
  Name string
  Type string
  Difficulty int
}

func fetchGymContests() []Contest {
  interrupt := make(chan bool)
  go runSpinner("fetching contests", interrupt)

  response, _ := http.Get("https://codeforces.com/api/contest.list?gym=true")
  body, _ := io.ReadAll(response.Body)

  interrupt <- true

  var fetchedObject ContestObject
  json.Unmarshal(body, &fetchedObject)

  if len(fetchedObject.Result) == 0 {
    fmt.Println("\rfetch contest failed!      ")
  } else {
    fmt.Println("\rdone!                      ")
  }

  return fetchedObject.Result
}

type ContestStatus struct {
  ContestId int
  ContestName string
  Verdicts map[string]string
}

func getGymContestStatus(handle string) map[int]ContestStatus {
  contestIdToName := make(map[int]string)
  for _, cont := range fetchGymContests() {
    contestIdToName[cont.Id] = cont.Name
  }

  submissions := fetchSubmission(handle)
  contestStatus := make(map[int]ContestStatus)
  for _, sub := range submissions {
    if (sub.ContestId < 10000) {
      continue
    }

    if _, exist := contestStatus[sub.ContestId]; !exist {
      tmp := ContestStatus { ContestId : sub.ContestId, ContestName: contestIdToName[sub.ContestId], Verdicts: make(map[string]string) }
      contestStatus[sub.ContestId] = tmp
    }

    if sub.Verdict == "OK" {
      contestStatus[sub.ContestId].Verdicts[sub.Problem.Index] = "O";
    } else if contestStatus[sub.ContestId].Verdicts[sub.Problem.Index] != "O" {
      contestStatus[sub.ContestId].Verdicts[sub.Problem.Index] = "X"
    }
  }

  return contestStatus
}

const rowsPerPage = 20
func printContestTable(contestStatus map[int]ContestStatus, pageNumber int) {
  firstLine := make([]string, 27)
  firstLine[0] = "Contest Name"
  for i := 1; i < 27; i++ {
    firstLine[i] = string('A' + (i - 1))
  }

  keys := make([]int, 0)
  for key, _ := range contestStatus {
    keys = append(keys, key)
  }

  sort.Ints(keys)
  slices.Reverse(keys)

  tbl := table.New(firstLine)
  for i := pageNumber * rowsPerPage; i < (pageNumber + 1) * rowsPerPage && i < len(keys); i++ {
    con := contestStatus[keys[i]]
    tbl.AddRow(termlink.Link(con.ContestName[:min(100, len(con.ContestName))], "https://codeforces.com/gym/" + strconv.Itoa(con.ContestId)))
    currentLine := make([]string, 27)
    currentLine[0] = "            "
    for i := 1; i < 27; i++ {
      if _, exist := con.Verdicts[string('A' + (i - 1))]; exist {
        currentLine[i] = con.Verdicts[string('A' + (i - 1))]
      } else {
        currentLine[i] = "-"
      }
    }
    tbl.AddRow(currentLine)
  }

  for i := len(keys); i < (pageNumber + 1) * rowsPerPage; i++ {
    tbl.AddRow()
    tbl.AddRow()
  }

  tbl.AddRow("page " + strconv.Itoa(pageNumber) + "/" + strconv.Itoa(len(contestStatus) / rowsPerPage))

  tbl.Print()
}

func printHelpTable() {
  fmt.Println("Commands:")
  fmt.Println("  :n - Show next page")
  fmt.Println("  :p - Show previous page")
  fmt.Println("  :h - Show help page")
  fmt.Println("  :q - Quit the process")
}

func main() {
  var handle string
  fmt.Printf("enter codeforces handle: ")
  fmt.Scan(&handle)
  contests := getGymContestStatus(handle)

  page := 0
  pageLimit := (len(contests) - 1) / rowsPerPage
  fmt.Println("\033[2J")
  printContestTable(contests, page)
  for {
    var command string
    fmt.Printf(":")
    fmt.Scan(&command)
    fmt.Println("\033[2J")
    switch {
    case command == "n":
      if page + 1 <= pageLimit {
        page = page + 1
        printContestTable(contests, page)
      } else {
        fmt.Println("already at the last page!")
      }
    case command == "p":
      if page - 1 >= 0 {
        page = page - 1
        printContestTable(contests, page)
      } else {
        fmt.Println("already at the first page!")
      }
    case command == "q":
      return
    case command == "h":
      printHelpTable()
    default:
      fmt.Println("unknown command, enter h to get some help.")
    }
  }
}
