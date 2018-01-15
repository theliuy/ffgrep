ffgrep - parallel file pattern searcher

  -c int
    	Set the maxiumn number of CPU when executing. (default 4)
  -e string
    	Match each line by the given regular expression pattern.
  -m int
    	Run m jobs to consume lines of the file. (default 4)
  -r int
    	Run up to r readers to read the file. (default 1)

Example:
  # search text pattern
  ffgrep "hello" access.log

  # search regular expression pattern
  ffgrep -e 'hello[ab]+world' access.log