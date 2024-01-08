/*

| Operation | Channel state      | Result                                                                                         |
|-----------|--------------------|------------------------------------------------------------------------------------------------|
| Read      | nil                | Block                                                                                          |
|           | Open and Not Empty | Value                                                                                          |
|           | Open and Empty     | Block                                                                                          |
|           | Closed             | \<default value>, false                                                                        |
|           | Write Only         | Compilation Error                                                                              |
| Write     | nil                | Block                                                                                          |
|           | Open and Full      | Block                                                                                          |
|           | Open and Not Full  | Write Value                                                                                    |
|           | Closed             | **panic**                                                                                      |
|           | Receive Only       | Compilation Error                                                                              |
| close     | nil                | panic                                                                                          |
|           | Open and Not Empty | Closes Channel; reads succeed until channel is drained, <br/> then reads produce default value |
|           | Open and Empty     | Closes Channel; reads produces default value                                                   |
|           | Closed             | **panic**                                                                                      |
|           | Receive Only       | Compilation Error                                                                              |
*/

package concurrency
