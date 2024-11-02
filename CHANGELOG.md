# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Add CORS support
### Fixed
- Fix PollResult voted bug
- Move GET request bodies to query for web

## [1.8.2] - 2024-07-15
### Fixed 
- Unable to load group polls [#66](https://github.com/rokwire/polls-building-block/issues/66)

## [1.8.1] - 2024-07-12
### Fixed 
- Crash when loading polls [#62](https://github.com/rokwire/polls-building-block/issues/62)

## [1.8.0] - 2024-07-09
### Added
- Remove user data [#59](https://github.com/rokwire/polls-building-block/issues/59)

## [1.7.1] - 2024-07-02
### Fixed 
- Broken normal poll notifications [#57](https://github.com/rokwire/polls-building-block/issues/57)

## [1.7.0] - 2024-07-01
### Changed
- Updated container images
- Update poll notifications [#55](https://github.com/rokwire/polls-building-block/issues/55)

## [1.6.0] - 2023-01-13
### Changed
- Update group's date update when a linked poll is created or ended [#49](https://github.com/rokwire/polls-building-block/issues/49)

## [1.5.1] - 2023-01-05
### Fixed 
- Confusing Error message on deleting a poll [#47](https://github.com/rokwire/polls-building-block/issues/47)

## [1.5.0] - 2022-12-06
### Added
- Delete survey responses api [#44](https://github.com/rokwire/polls-building-block/issues/43)

## [1.4.0] - 2022-11-18
### Added
- Improve survey models for BESSI [#40](https://github.com/rokwire/polls-building-block/issues/40)
- Send notifications for group polls directly through Groups BB [#30](https://github.com/rokwire/polls-building-block/issues/30)
- Set appID and orgID for notifications [#26](https://github.com/rokwire/polls-building-block/issues/26)
- Set logger [#11](https://github.com/rokwire/polls-building-block/issues/11)

## [1.3.0] - 2022-11-10
### Added
- Post survey alert and CRUD alert contact apis [#32](https://github.com/rokwire/polls-building-block/issues/32)

## [1.2.0] - 2022-10-30
### Added
- Get survey responses for a user [#34](https://github.com/rokwire/polls-building-block/issues/34)

## [1.1.0] - 2022-10-26
### Added
- Created CRUD APIs for Surveys [#23](https://github.com/rokwire/polls-building-block/issues/23)
- Created CRUD APIs for Survey Responses [#24](https://github.com/rokwire/polls-building-block/issues/24)

## Fixed
- Fix docs [#21](https://github.com/rokwire/polls-building-block/issues/21)

## [1.0.21] - 2022-08-11
### Fixed
- Update poll ended notification text [#18](https://github.com/rokwire/polls-building-block/issues/18)
- Fix detect-secrets and Makefile [#16](https://github.com/rokwire/polls-building-block/issues/16)

## [1.0.20] - 2022-07-28
### Fixed
- Allow Group Admins to start/end/delete a group poll [#13](https://github.com/rokwire/polls-building-block/issues/13)

## [1.0.19] - 2022-06-03
### Fixed
- Allow group admins to delete or end polls [#9](https://github.com/rokwire/polls-building-block/issues/9)
- Only creator can edit or delete a poll  [#9](https://github.com/rokwire/polls-building-block/issues/9)
- Additional fix for notifying group members & group sub members with respect to the admins [#9](https://github.com/rokwire/polls-building-block/issues/9)

## [1.0.18] - 2022-06-02
### Fixed
- Fix broken poll vote [#9](https://github.com/rokwire/polls-building-block/issues/9)
- Additional fix for group admin should see all private polls [#9](https://github.com/rokwire/polls-building-block/issues/9)

## [1.0.17] - 2022-06-01
### Fixed
- Additional fixes for polls and the integration with groups and notifications [#9](https://github.com/rokwire/polls-building-block/issues/9)

## [1.0.16] - 2022-05-31
### Changed
- Rework poll notifications to participants & group members [#9](https://github.com/rokwire/polls-building-block/issues/9)

## [1.0.15] - 2022-05-26
### Added
- Prepare the project to become open source [#8](https://github.com/rokwire/polls-building-block/issues/8)
- Add more logic for supporting group polls and cover more edge cases [#9](https://github.com/rokwire/polls-building-block/issues/9)

## [1.0.14] - 2022-05-20
### Added
- Add support of group polls and subgroups [#9](https://github.com/rokwire/polls-building-block/issues/9)

## [1.0.13] - 2022-05-04
### Added
- Implement internal API for retrieving of group polls for data migration to groups BB [#5](https://github.com/rokwire/polls-building-block/issues/5)

## [1.0.12] - 2022-04-29
### Changed
- Update core auth library to the latest version [#3](https://github.com/rokwire/polls-building-block/issues/3)

## [1.0.11] - 2022-04-26
### Changed
- Update go to 1.18, alpine 3.15[#1](https://github.com/rokwire/polls-building-block/issues/1)
- Resolve [GHSA-xg75-q3q5-cqmv](https://github.com/advisories/GHSA-xg75-q3q5-cqmv) [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.10] - 2022-04-21
### Added
- Implemented additional filter for responded_polls[#1](https://github.com/rokwire/polls-building-block/issues/1)
- Additional fixes for the event stream [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.9] - 2022-04-20
### Added
- Fixed ID generation in the create new poll API [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.8] - 2022-04-19
### Added
- Additional filtering by pin and group_ids and API fixes [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.7] - 2022-04-18
### Added
- Rework GetPolls filtering (add more filter options in the request body) [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.6] - 2022-04-15
### Added
- Implemented support for SSE [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.5] - 2022-04-11
### Added
- Implemented support for subgroups [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.4] - 2022-04-07
### Added
- Additional improvements of the polls APIs [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.4] - 2022-04-07
### Added
- Additional improvements of the polls APIs [#1](https://github.com/rokwire/polls-building-block/issues/1)

## [1.0.1] - 2022-04-06
### Added
- Introduce Polls BB 