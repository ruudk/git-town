Feature: already existing branch

  Scenario Outline:
    Given a <LOCATION> feature branch "existing"
    When I run "git-town hack existing"
    Then it runs the commands
      | BRANCH | COMMAND                  |
      | main   | git fetch --prune --tags |
    And it prints the error:
      """
      there is already a branch "existing"
      """

    Examples:
      | LOCATION |
      | local    |
      | remote   |
