### Description

<!--
  The description should provide all necessary information for a reviewer.
  - What does this PR change, what's the reason for the change, how can it be tested
-->
&lt;Add detailed description for reviewers here.&gt;


### Submitter checklist

- [ ] Change has been tested (on a back-end cluster)
- [ ] (If applicable) Jira backlog item ID added to the PR title and the [changelog.yaml] entry
- [ ] [changelog.yaml] entry with upgrade notes is prepared and appropriate for the audience affected by the change (users or developer, depending on the change)
- [ ] Semantic version diffed against [last release][releases] and updated accordingly. In this project the version has to be maintained here:
    - [/charts/steward/Chart.yaml](https://github.com/SAP/stewardci-core/blob/master/charts/steward/Chart.yaml) (`version` and `appVersion`)

In case dependencies have been updated:
- [ ] Links to external changelogs, since the last release of our component, added to the [changelog.yaml] entry (description).
- [ ] Changelogs read thoroughly, potential impact described, upgrade notes prepared (if necessary)
- [ ] Check if dependency updates affect our semantic version increment.

### Reviewer checklist

Before the changes are marked as `ready-for-merge`: 

- [ ] There is at least 1 approval for the pull request and no outstanding requests for change
- [ ] All voter checks have passed
- [ ] Conversations in the pull request are over OR it is explicit that a reviewer does not block the change
- [ ] The Pull Request title is understandable and reflects the changes well
- [ ] The Pull Request description is understandable and well documented
- [ ] [changelog.yaml] entry for this Pull Request has been added
    - [ ] Changelog entry contains all required information
    - [ ] 'Upgrade notes' are documented in changelog.yaml (if required)

[changelog.yaml]: https://github.com/SAP/stewardci-core/changelog.yaml
[releases]: https://github.com/SAP/stewardci-core/releases
