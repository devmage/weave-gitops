import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from "@material-ui/core";
import _ from "lodash";
import * as React from "react";
import styled from "styled-components";
import Button from "./Button";
import Flex from "./Flex";
import Icon, { IconType } from "./Icon";
import Spacer from "./Spacer";
import Text from "./Text";

type Sorter = (k: any) => any;

export type Field = {
  label: string | number;
  labelRenderer?: string | ((k: any) => string | JSX.Element);
  value: string | ((k: any) => string | JSX.Element | null);
  sortValue?: Sorter;
  textSearchable?: boolean;
  maxWidth?: number;
  /** boolean for field to initially sort against. */
  defaultSort?: boolean;
  /** boolean for field to implement secondary sort against. */
  secondarySort?: boolean;
};

/** DataTable Properties  */
export interface Props {
  /** CSS MUI Overrides or other styling. */
  className?: string;
  /** A list of objects with four fields: `label`, which is a string representing the column header, `value`, which can be a string, or a function that extracts the data needed to fill the table cell, and `sortValue`, which customizes your input to the search function */
  fields: Field[];
  /** A list of data that will be iterated through to create the columns described in `fields`. */
  rows?: any[];
  /** for passing pagination */
  children?: any;
}

const EmptyRow = styled(TableRow)<{ colSpan: number }>`
  td {
    text-align: center;
  }
`;

const TableButton = styled(Button)`
  &.MuiButton-root {
    margin: 0;
    text-transform: none;
    letter-spacing: 0;
  }
  &.MuiButton-text {
    min-width: 0px;
    .selected {
      color: ${(props) => props.theme.colors.neutral40};
    }
  }
  &.arrow {
    min-width: 0px;
  }
`;

type Row = any;

export const sortByField = (
  rows: Row[],
  reverseSort: boolean,
  sortFields: Field[],
  useSecondarySort?: boolean
) => {
  const orderFields = [
    sortFields[0],
    ...(useSecondarySort && sortFields.length > 1 && [sortFields[1]]),
  ];

  return _.orderBy(
    rows,
    sortFields.map((s) => {
      return s.sortValue || s.value;
    }),
    orderFields.map((_, index) => {
      // Always sort secondary sort values in the ascending order.
      const sortOrders =
        reverseSort && (!useSecondarySort || index != 1) ? "desc" : "asc";

      return sortOrders;
    })
  );
};

type labelProps = {
  fields: Field[];
  fieldIndex: number;
  sortFieldIndex: number;
  reverseSort: boolean;
  setSortFieldIndex: (index: number) => void;
  setReverseSort: (b: boolean) => void;
};

function SortableLabel({
  fields,
  fieldIndex,
  sortFieldIndex,
  reverseSort,
  setSortFieldIndex,
  setReverseSort,
}: labelProps) {
  const field = fields[fieldIndex];
  const sort = fields[sortFieldIndex];

  return (
    <Flex align start>
      <TableButton
        color="inherit"
        variant="text"
        onClick={() => {
          setReverseSort(sortFieldIndex === fieldIndex ? !reverseSort : false);
          setSortFieldIndex(fieldIndex);
        }}
      >
        <h2 className={sort.label === field.label ? "selected" : ""}>
          {field.label}
        </h2>
      </TableButton>
      <Spacer padding="xxs" />
      {sort.label === field.label ? (
        <Icon
          type={IconType.ArrowUpwardIcon}
          size="base"
          className={reverseSort ? "upward" : "downward"}
        />
      ) : (
        <div style={{ width: "16px" }} />
      )}
    </Flex>
  );
}

/** Form DataTable */
function UnstyledDataTable({ className, fields, rows, children }: Props) {
  const [sortFieldIndex, setSortFieldIndex] = React.useState(() => {
    let sortFieldIndex = fields.findIndex((f) => f.defaultSort);

    if (sortFieldIndex === -1) {
      sortFieldIndex = 0;
    }

    return sortFieldIndex;
  });

  const secondarySortFieldIndex = fields.findIndex((f) => f.secondarySort);

  const [reverseSort, setReverseSort] = React.useState(false);

  let sortFields = [fields[sortFieldIndex]];

  const useSecondarySort =
    secondarySortFieldIndex > -1 && sortFieldIndex != secondarySortFieldIndex;

  if (useSecondarySort) {
    sortFields = sortFields.concat(fields[secondarySortFieldIndex]);
    sortFields = sortFields.concat(
      fields.filter(
        (_, index) =>
          index != sortFieldIndex && index != secondarySortFieldIndex
      )
    );
  } else {
    sortFields = sortFields.concat(
      fields.filter((_, index) => index != sortFieldIndex)
    );
  }

  const sorted = sortByField(rows, reverseSort, sortFields, useSecondarySort);

  const r = _.map(sorted, (r, i) => (
    <TableRow key={i}>
      {_.map(fields, (f) => (
        <TableCell
          style={
            f.maxWidth && {
              maxWidth: f.maxWidth,
            }
          }
          key={f.label}
        >
          <Text>{typeof f.value === "function" ? f.value(r) : r[f.value]}</Text>
        </TableCell>
      ))}
    </TableRow>
  ));

  return (
    <div className={className}>
      <TableContainer>
        <Table aria-label="simple table">
          <TableHead>
            <TableRow>
              {_.map(fields, (f, index) => (
                <TableCell key={f.label}>
                  {typeof f.labelRenderer === "function" ? (
                    f.labelRenderer(r)
                  ) : (
                    <SortableLabel
                      fields={fields}
                      fieldIndex={index}
                      sortFieldIndex={sortFieldIndex}
                      reverseSort={reverseSort}
                      setSortFieldIndex={setSortFieldIndex}
                      setReverseSort={(isReverse) => setReverseSort(isReverse)}
                    />
                  )}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {r.length > 0 ? (
              r
            ) : (
              <EmptyRow colSpan={fields.length}>
                <TableCell colSpan={fields.length}>
                  <Flex center align>
                    <Icon
                      color="neutral20"
                      type={IconType.RemoveCircleIcon}
                      size="base"
                    />
                    <Spacer padding="xxs" />
                    <Text color="neutral30">No data</Text>
                  </Flex>
                </TableCell>
              </EmptyRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      <Spacer padding="xs" />
      {/* optional pagination component */}
      {children}
    </div>
  );
}

export const DataTable = styled(UnstyledDataTable)`
  width: 100%;
  flex-wrap: nowrap;
  overflow-x: auto;
  h2 {
    padding: ${(props) => props.theme.spacing.xs};
    font-size: 14px;
    font-weight: 600;
    color: ${(props) => props.theme.colors.neutral30};
    margin: 0px;
    white-space: nowrap;
  }
  .MuiTableRow-root {
    transition: background 0.5s ease-in-out;
  }
  .MuiTableRow-root:not(.MuiTableRow-head):hover {
    background: ${(props) => props.theme.colors.neutral10};
    transition: background 0.5s ease-in-out;
  }
  table {
    margin-top: ${(props) => props.theme.spacing.small};
  }
  th {
    padding: ${(props) => props.theme.spacing.xs};
    background: ${(props) => props.theme.colors.neutralGray};
  }
  td {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
`;

export default DataTable;
